// Package ai runs an LLM agent (pi-go) over the music library, reading data
// through tools rather than stuffing the library into the prompt. One model and
// toolset back radio, the Ask chat surface, and Wrapped.
package ai

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"

	"github.com/Aunali321/korus/internal/services"
)

// DefaultModel is used when AI_MODEL is unset.
const DefaultModel = "arcee-ai/trinity-large-thinking:arcee-ai"

// maxRetries bounds pi-go's retry of the initial provider request. OpenRouter
// throttles with 429s that carry a retry-after, which is worth waiting out
// rather than failing a whole radio or chat turn.
const maxRetries = 3

// discoveryTimeout bounds the catalog fetch so an unreachable OpenRouter
// cannot hold up server startup. The catalog is a few megabytes and this runs
// once per process, so the limit is generous rather than tight.
const discoveryTimeout = 15 * time.Second

// Config wires the AI service to the data services it reads and writes
// through. It owns no music SQL of its own; db is only for the AI's own tables
// and the persisted player state.
type Config struct {
	DB        *sql.DB
	Library   *services.LibraryService
	Playlists *services.PlaylistService
	Search    *services.SearchService
	Stats     *services.StatsService
	APIKey    string
	Provider  string
	BaseURL   string
	Model     string
	Reasoning string
}

type Service struct {
	db        *sql.DB
	library   *services.LibraryService
	playlists *services.PlaylistService
	search    *services.SearchService
	stats     *services.StatsService
	apiKey    string
	model     *llm.Model
	reasoning llm.ThinkingLevel
}

func New(cfg Config) *Service {
	modelID, provider, baseURL := cfg.Model, cfg.Provider, cfg.BaseURL
	if modelID == "" {
		modelID = DefaultModel
	}
	if provider == "" {
		provider = "openrouter"
	}
	model := &llm.Model{
		ID:            modelID,
		Name:          modelID,
		Provider:      provider,
		BaseURL:       baseURL, // empty → pi-go defaults to OpenRouter
		Reasoning:     true,
		Input:         []llm.InputModality{llm.InputText},
		ContextWindow: 128000,
		MaxTokens:     65536,
	}
	if provider == "openrouter" {
		applyCatalog(model, baseURL)
	}
	return &Service{
		db:        cfg.DB,
		library:   cfg.Library,
		playlists: cfg.Playlists,
		search:    cfg.Search,
		stats:     cfg.Stats,
		apiKey:    cfg.APIKey,
		model:     model,
		reasoning: parseReasoning(cfg.Reasoning),
	}
}

// applyCatalog replaces the conservative built-in metadata with the model's
// real context window, token ceiling, pricing and thinking levels from
// OpenRouter's catalog. The configured ID is kept because it may carry a
// ":provider" routing suffix the catalog does not list, so the catalog entry is
// matched on the base ID too. Discovery is best-effort: a failed fetch or an
// unlisted model leaves the built-in values rather than stopping the server.
func applyCatalog(model *llm.Model, baseURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	catalog, err := llm.FetchOpenRouterModels(ctx, nil)
	if err != nil {
		log.Printf("ai: OpenRouter model discovery failed (%v); using built-in metadata for %s", err, model.ID)
		return
	}

	base, _, _ := strings.Cut(model.ID, ":")
	var match *llm.Model
	for _, cm := range catalog {
		if cm.ID == model.ID {
			match = cm
			break
		}
		if cm.ID == base && match == nil {
			match = cm
		}
	}
	if match == nil {
		log.Printf("ai: %s is not in the OpenRouter catalog; using built-in metadata", model.ID)
		return
	}

	id := model.ID
	*model = *match
	model.ID = id
	if baseURL != "" {
		model.BaseURL = baseURL
	}
	log.Printf("ai: %s context=%d max_tokens=%d reasoning=%v", model.ID, model.ContextWindow, model.MaxTokens, llm.SupportedThinkingLevels(model))
}

func (s *Service) ModelID() string { return s.model.ID }

// debugSink logs tool calls and final text to stderr when AI_DEBUG is set.
func debugSink() agent.EventSink {
	if os.Getenv("AI_DEBUG") == "" {
		return func(agent.Event) {}
	}
	return func(e agent.Event) {
		switch ev := e.(type) {
		case agent.ToolExecutionStart:
			log.Printf("[ai] → %s %v", ev.ToolName, ev.Args)
		case agent.ToolExecutionEnd:
			log.Printf("[ai] ← %s (error=%v)", ev.ToolName, ev.IsError)
		case agent.AgentEnd:
			for _, m := range ev.Messages {
				if am, ok := m.(*llm.AssistantMessage); ok {
					for _, c := range am.Content {
						if t, ok := c.(*llm.Text); ok && t.Text != "" {
							log.Printf("[ai] assistant: %s", t.Text)
						}
					}
				}
			}
		}
	}
}

func parseReasoning(s string) llm.ThinkingLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off", "none":
		return llm.ThinkingOff
	case "minimal":
		return llm.ThinkingMinimal
	case "low":
		return llm.ThinkingLow
	case "high":
		return llm.ThinkingHigh
	case "xhigh":
		return llm.ThinkingXHigh
	case "max":
		return llm.ThinkingMax
	default:
		return llm.ThinkingMedium
	}
}

// streamOptions are shared by every agent run.
func (s *Service) streamOptions() llm.StreamOptions {
	return llm.StreamOptions{
		APIKey:         s.apiKey,
		CacheRetention: llm.CacheShort,
		MaxTokens:      s.model.MaxTokens,
		MaxRetries:     maxRetries,
	}
}

// runError reports the first failed assistant message in a finished run.
func runError(msgs []agent.AgentMessage, failed string) error {
	for _, m := range msgs {
		if am, ok := m.(*llm.AssistantMessage); ok && (am.StopReason == llm.StopError || am.StopReason == llm.StopAborted) {
			if am.ErrorMessage != "" {
				return errors.New(am.ErrorMessage)
			}
			return errors.New(failed)
		}
	}
	return nil
}

// run drives a one-shot agent until it stops or a tool terminates the loop.
func (s *Service) run(ctx context.Context, systemPrompt, userPrompt string, tools []agent.Tool, emit agent.EventSink) ([]agent.AgentMessage, error) {
	agentCtx := &agent.Context{SystemPrompt: systemPrompt, Tools: tools}
	cfg := &agent.Config{
		Model:     s.model,
		Reasoning: s.reasoning,
		Options:   s.streamOptions(),
	}
	if emit == nil {
		emit = debugSink()
	}
	msgs := agent.Run(ctx, []agent.AgentMessage{llm.TextUser(userPrompt)}, agentCtx, cfg, emit)
	return msgs, runError(msgs, "ai: agent run failed")
}
