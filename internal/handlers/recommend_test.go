package handlers

import (
	"testing"

	"korus/internal/config"
	"korus/internal/services"
)

func TestRecommendHandlerParseLimit(t *testing.T) {
	h := NewRecommendHandler(&services.RecommenderService{}, &config.RecommenderConfig{SimilarityLimit: 50})

	if got := h.parseLimit("25"); got != 25 {
		t.Fatalf("expected 25, got %d", got)
	}
	if got := h.parseLimit("-3"); got != 50 {
		t.Fatalf("expected default 50, got %d", got)
	}
	if got := h.parseLimit("500"); got != 50 {
		t.Fatalf("expected capped 50, got %d", got)
	}
}
