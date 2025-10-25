package jobs

import (
	"encoding/json"
	"testing"

	"korus/internal/models"
)

func TestUnmarshalEmbeddingPayload(t *testing.T) {
	queue := &Queue{}
	ptr := func(v int) *int { return &v }

	payload := EmbeddingExtractJobPayload{FilePath: "song.mp3", SongID: ptr(42)}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	job := &Job{Job: &models.Job{JobType: JobTypeEmbeddingExtract, Payload: data}}
	if err := queue.unmarshalPayload(job); err != nil {
		t.Fatalf("unmarshalPayload returned error: %v", err)
	}

	decoded, ok := job.PayloadData.(EmbeddingExtractJobPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", job.PayloadData)
	}
	if decoded.FilePath != payload.FilePath {
		t.Fatalf("expected file path %s, got %s", payload.FilePath, decoded.FilePath)
	}
	if decoded.SongID == nil || *decoded.SongID != 42 {
		t.Fatalf("expected song id 42, got %v", decoded.SongID)
	}
}

func TestUnmarshalBatchEmbeddingPayload(t *testing.T) {
	queue := &Queue{}
	ptr := func(v int) *int { return &v }

	batch := BatchEmbeddingExtractJobPayload{
		Entries: []BatchEmbeddingEntry{
			{FilePath: "a.mp3", SongID: ptr(1)},
			{FilePath: "b.mp3", SongID: ptr(2)},
		},
	}

	data, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("failed to marshal batch payload: %v", err)
	}

	job := &Job{Job: &models.Job{JobType: JobTypeEmbeddingExtractBatch, Payload: data}}
	if err := queue.unmarshalPayload(job); err != nil {
		t.Fatalf("unmarshalPayload returned error: %v", err)
	}

	decoded, ok := job.PayloadData.(BatchEmbeddingExtractJobPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", job.PayloadData)
	}
	if len(decoded.Entries) != len(batch.Entries) {
		t.Fatalf("expected %d entries, got %d", len(batch.Entries), len(decoded.Entries))
	}
	for i, entry := range decoded.Entries {
		if entry.FilePath != batch.Entries[i].FilePath {
			t.Fatalf("entry %d expected path %s, got %s", i, batch.Entries[i].FilePath, entry.FilePath)
		}
		if entry.SongID == nil || *entry.SongID != *batch.Entries[i].SongID {
			t.Fatalf("entry %d expected song id %d, got %v", i, *batch.Entries[i].SongID, entry.SongID)
		}
	}
}
