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

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"
)

// DefaultModel is used when AI_MODEL is unset.
const DefaultModel = "arcee-ai/trinity-large-thinking:arcee-ai"

type Service struct {
	db        *sql.DB
	apiKey    string
	model     *llm.Model
	reasoning llm.ThinkingLevel
}

func New(db *sql.DB, apiKey, provider, baseURL, modelID, reasoning string) *Service {
	if modelID == "" {
		modelID = DefaultModel
	}
	if provider == "" {
		provider = "openrouter"
	}
	return &Service{
		db:     db,
		apiKey: apiKey,
		model: &llm.Model{
			ID:            modelID,
			Name:          modelID,
			Provider:      provider,
			BaseURL:       baseURL, // empty → pi-go defaults to OpenRouter
			Reasoning:     true,
			Input:         []llm.InputModality{llm.InputText},
			ContextWindow: 128000,
			MaxTokens:     65536,
		},
		reasoning: parseReasoning(reasoning),
	}
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
	default:
		return llm.ThinkingMedium
	}
}

// run drives a one-shot agent until it stops or a tool terminates the loop.
func (s *Service) run(ctx context.Context, systemPrompt, userPrompt string, tools []agent.Tool, emit agent.EventSink) ([]agent.AgentMessage, error) {
	agentCtx := &agent.Context{SystemPrompt: systemPrompt, Tools: tools}
	cfg := &agent.Config{
		Model:     s.model,
		Reasoning: s.reasoning,
		Options: llm.StreamOptions{
			APIKey:         s.apiKey,
			CacheRetention: llm.CacheShort,
			MaxTokens:      s.model.MaxTokens,
		},
	}
	if emit == nil {
		emit = debugSink()
	}
	msgs := agent.Run(ctx, []agent.AgentMessage{llm.TextUser(userPrompt)}, agentCtx, cfg, emit)
	for _, m := range msgs {
		if am, ok := m.(*llm.AssistantMessage); ok && (am.StopReason == llm.StopError || am.StopReason == llm.StopAborted) {
			if am.ErrorMessage != "" {
				return msgs, errors.New(am.ErrorMessage)
			}
			return msgs, errors.New("ai: agent run failed")
		}
	}
	return msgs, nil
}
