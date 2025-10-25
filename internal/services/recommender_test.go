package services

import (
	"math"
	"testing"
)

func TestComputeSimilarity(t *testing.T) {
	vecA := []float32{3, 4, 1, 0}
	vecB := []float32{4, 3, 0, 1}

	normalizeBlocks(vecA, 2)
	normalizeBlocks(vecB, 2)

	recA := &embeddingRecord{SongID: 1, Vector: vecA, Dim: 2, Segments: 1}
	recB := &embeddingRecord{SongID: 2, Vector: vecB, Dim: 2, Segments: 1}

	score := computeSimilarity(recA, recB)
	if score <= 0 {
		t.Fatalf("expected positive similarity, got %f", score)
	}
	if score > 1.0 {
		t.Fatalf("expected similarity <= 1, got %f", score)
	}
}

func TestNormalizeBlocks(t *testing.T) {
	vec := []float32{3, 4, 5, 12}
	normalizeBlocks(vec, 2)

	block1 := math.Sqrt(float64(vec[0]*vec[0] + vec[1]*vec[1]))
	block2 := math.Sqrt(float64(vec[2]*vec[2] + vec[3]*vec[3]))

	if math.Abs(block1-1.0) > 1e-6 {
		t.Fatalf("expected block1 norm 1, got %f", block1)
	}
	if math.Abs(block2-1.0) > 1e-6 {
		t.Fatalf("expected block2 norm 1, got %f", block2)
	}
}

func TestMinInt(t *testing.T) {
	if minInt(5, 9) != 5 {
		t.Fatalf("minInt expected 5")
	}
	if minInt(-3, -7) != -7 {
		t.Fatalf("minInt expected -7")
	}
}
