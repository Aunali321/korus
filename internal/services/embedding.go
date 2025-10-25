package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"korus/internal/config"
	"korus/internal/database"
)

type EmbeddingTask struct {
	SongID   *int
	FilePath string
}

type embeddingEntry struct {
	SongID   int
	FilePath string
}

type ProcessedEmbedding struct {
	SongID   int
	FilePath string
}

type embeddingResult struct {
	Path     string    `json:"path"`
	Vector   []float64 `json:"vector"`
	Dim      int       `json:"dim"`
	Segments int       `json:"segments"`
	Method   string    `json:"method"`
	Model    string    `json:"model"`
}

type EmbeddingService struct {
	db                *database.DB
	cfg               *config.RecommenderConfig
	runner            []string
	scriptPath        string
	projectDir        string
	encoderConfigPath string
}

func NewEmbeddingService(db *database.DB, cfg *config.RecommenderConfig) (*EmbeddingService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("recommender config is nil")
	}

	service := &EmbeddingService{db: db, cfg: cfg}

	if !cfg.Enabled {
		return service, nil
	}

	runnerParts := strings.Fields(cfg.PythonRunner)
	if len(runnerParts) == 0 {
		return nil, fmt.Errorf("invalid python runner configuration")
	}

	absoluteProjectDir, err := filepath.Abs(cfg.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project directory: %w", err)
	}

	scriptPath := filepath.Join(absoluteProjectDir, "embed.py")
	if _, statErr := os.Stat(scriptPath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return nil, fmt.Errorf("embedding script not found at %s", scriptPath)
		}
		return nil, fmt.Errorf("failed to access embedding script: %w", statErr)
	}

	encoderConfigPath := cfg.EncoderConfigPath
	if encoderConfigPath == "" {
		// Default path should be relative to the Python project directory
		encoderConfigPath = "configs/LongCatAudioCodec_encoder.yaml"
	}

	// For relative paths, we need to resolve them relative to the Python project directory
	// since that's where the Python script will be running from
	var resolvedConfig string
	if filepath.IsAbs(encoderConfigPath) {
		// Absolute path - use as-is
		resolvedConfig = filepath.Clean(encoderConfigPath)
		if info, err := os.Stat(resolvedConfig); err != nil || info.IsDir() {
			resolvedConfig = ""
		}
	} else {
		// Relative path - resolve relative to the Python project directory
		candidatePaths := []string{
			filepath.Join(absoluteProjectDir, encoderConfigPath), // Relative to Python project dir
		}

		// Also try resolving relative to the current working directory for backward compatibility
		if absPath, err := filepath.Abs(encoderConfigPath); err == nil {
			candidatePaths = append(candidatePaths, absPath)
		}

		// Try the original path as-is (in case it's already correctly resolved)
		candidatePaths = append(candidatePaths, encoderConfigPath)

		for _, candidate := range candidatePaths {
			clean := filepath.Clean(candidate)
			if info, err := os.Stat(clean); err == nil && !info.IsDir() {
				resolvedConfig = clean
				break
			}
		}
	}

	if resolvedConfig == "" {
		return nil, fmt.Errorf("encoder config not found; tried path: %s (absolute: %s, relative to project: %s)",
			encoderConfigPath, absoluteProjectDir, filepath.Join(absoluteProjectDir, encoderConfigPath))
	}

	// Convert the resolved config to be relative to the Python project directory for the script
	finalConfigPath := encoderConfigPath
	if filepath.IsAbs(resolvedConfig) {
		// If we resolved to an absolute path, convert it back to relative for the Python script
		relPath, err := filepath.Rel(absoluteProjectDir, resolvedConfig)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			finalConfigPath = relPath
		} else {
			// Fallback to absolute path if we can't make it relative
			finalConfigPath = resolvedConfig
		}
	} else {
		finalConfigPath = resolvedConfig
	}

	service.runner = runnerParts
	service.scriptPath = scriptPath
	service.projectDir = absoluteProjectDir
	service.encoderConfigPath = finalConfigPath

	return service, nil
}

func (s *EmbeddingService) Enabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Enabled
}

func (s *EmbeddingService) ProcessBatch(ctx context.Context, tasks []EmbeddingTask) ([]ProcessedEmbedding, error) {
	processed := make([]ProcessedEmbedding, 0, len(tasks))

	if !s.Enabled() || len(tasks) == 0 {
		return processed, nil
	}

	entriesByPath := make(map[string][]embeddingEntry)
	orderedPaths := make([]string, 0, len(tasks))

	for _, task := range tasks {
		if task.FilePath == "" {
			continue
		}

		songID := task.SongID
		if songID == nil {
			id, err := s.lookupSongID(ctx, task.FilePath)
			if err != nil {
				if !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Failed to resolve song ID for %s: %v", task.FilePath, err)
				}
				continue
			}
			songID = &id
		}

		if songID == nil {
			log.Printf("Skipping embedding extraction for %s: song not found", task.FilePath)
			continue
		}

		taskEntry := embeddingEntry{SongID: *songID, FilePath: task.FilePath}
		if _, exists := entriesByPath[task.FilePath]; !exists {
			orderedPaths = append(orderedPaths, task.FilePath)
		}
		entriesByPath[task.FilePath] = append(entriesByPath[task.FilePath], taskEntry)
	}

	if len(orderedPaths) == 0 {
		log.Printf("No valid embedding tasks to process")
		return processed, nil
	}

	results, stderrOutput, err := s.runExtractor(ctx, orderedPaths)
	if err != nil {
		if stderrOutput != "" {
			log.Printf("Embedding extractor stderr: %s", stderrOutput)
		}
		return processed, err
	}

	missing := []string{}

	for path, entries := range entriesByPath {
		result, ok := results[path]
		if !ok {
			missing = append(missing, path)
			continue
		}

		for _, entry := range entries {
			if err := s.upsertEmbedding(ctx, entry.SongID, result); err != nil {
				return processed, fmt.Errorf("failed to store embedding for song %d: %w", entry.SongID, err)
			}
			processed = append(processed, ProcessedEmbedding{SongID: entry.SongID, FilePath: entry.FilePath})
		}
	}

	if len(missing) > 0 {
		return processed, fmt.Errorf("embedding extractor returned no data for %d files", len(missing))
	}

	return processed, nil
}

func (s *EmbeddingService) runExtractor(ctx context.Context, filePaths []string) (map[string]embeddingResult, string, error) {
	if len(filePaths) == 0 {
		return map[string]embeddingResult{}, "", nil
	}

	if len(s.runner) == 0 {
		return nil, "", fmt.Errorf("embedding runner is not configured")
	}

	cmdName := s.runner[0]
	cmdArgs := append([]string{}, s.runner[1:]...)

	cmdArgs = append(cmdArgs, s.scriptPath,
		"--encoder-config", s.encoderConfigPath,
		"--segments", strconv.Itoa(s.cfg.Segments),
		"--batch-size", strconv.Itoa(s.cfg.BatchSize),
		"--project-dim", strconv.Itoa(s.cfg.ProjectDim),
		"--method", "longcat-semantic-tpp-v1",
		"--files")
	cmdArgs = append(cmdArgs, filePaths...)

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Dir = s.projectDir
	env := append(os.Environ(), "PYTHONPATH="+s.projectDir)
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", fmt.Errorf("failed to capture stdout: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, "", fmt.Errorf("failed to capture stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("failed to start embedding extractor: %w", err)
	}

	stderrBuf := &bytes.Buffer{}
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(stderrBuf, stderr)
		close(stderrDone)
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024), 16*1024*1024)

	results := make(map[string]embeddingResult)
	for scanner.Scan() {
		line := scanner.Bytes()
		var result embeddingResult
		if err := json.Unmarshal(line, &result); err != nil {
			log.Printf("Failed to decode embedding output: %v", err)
			continue
		}
		results[result.Path] = result
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		<-stderrDone
		return nil, stderrBuf.String(), fmt.Errorf("failed to read embedding output: %w", err)
	}

	waitErr := cmd.Wait()
	<-stderrDone

	if waitErr != nil {
		return nil, stderrBuf.String(), fmt.Errorf("embedding extractor failed: %w", waitErr)
	}

	return results, stderrBuf.String(), nil
}

func (s *EmbeddingService) upsertEmbedding(ctx context.Context, songID int, result embeddingResult) error {
	if len(result.Vector) == 0 {
		return fmt.Errorf("empty embedding vector")
	}

	vec := make([]float32, len(result.Vector))
	for i, value := range result.Vector {
		vec[i] = float32(value)
	}

	query := `
		INSERT INTO song_embeddings (song_id, embedding, dim, segments, method, model, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (song_id)
		DO UPDATE SET
			embedding = EXCLUDED.embedding,
			dim = EXCLUDED.dim,
			segments = EXCLUDED.segments,
			method = EXCLUDED.method,
			model = EXCLUDED.model,
			updated_at = NOW()
	`

	floatArray := pgtype.Array[float32]{
		Elements: append([]float32(nil), vec...),
		Valid:    true,
		Dims:     []pgtype.ArrayDimension{{Length: int32(len(vec)), LowerBound: 1}},
	}

	_, err := s.db.ExecContext(ctx, query,
		songID,
		&floatArray,
		result.Dim,
		result.Segments,
		result.Method,
		result.Model,
	)
	return err
}

func (s *EmbeddingService) lookupSongID(ctx context.Context, filePath string) (int, error) {
	query := `SELECT id FROM songs WHERE file_path = $1`
	var id int
	err := s.db.QueryRowContext(ctx, query, filePath).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}
