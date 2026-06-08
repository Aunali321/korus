package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

const radioSystem = `You are a music curator building a "radio" queue from a seed song, drawn entirely from a self-hosted library you can only see through tools.

Assemble an ordered queue that flows naturally from the seed — similar genre, era, energy, and mood — biased toward what this listener actually enjoys.

- Use get_top_tracks and get_recent_plays to learn the listener's taste. Use search_songs and get_artist_songs to find candidates.
- Only pick songs that appear in tool results. Never invent IDs.
- Exclude the seed song and its alternate versions (remix, live, acoustic, cover).
- Keep it cohesive but varied; avoid the same artist back-to-back.

This runs non-interactively — never ask the user questions. Always finish by calling submit_songs exactly once with whatever fits: as many good matches as exist (up to the requested count), or an empty list if truly nothing fits.`

// Radio builds a personalized radio queue of song IDs seeded from seedID.
func (s *Service) Radio(ctx context.Context, userID, seedID int64, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 20
	}
	seed, err := s.songBriefByID(ctx, seedID)
	if err != nil {
		return nil, err
	}
	seedJSON, _ := json.Marshal(seed)

	var picked []int64
	tools := append(s.readTools(userID), submitSongIDsTool(&picked))
	prompt := fmt.Sprintf("Seed song:\n%s\n\nBuild a radio queue of about %d songs that flow from it.", string(seedJSON), limit)

	if _, err := s.run(ctx, radioSystem, prompt, tools, nil); err != nil {
		return nil, err
	}

	ids := make([]int64, 0, limit)
	seen := map[int64]bool{seedID: true}
	for _, id := range picked {
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		if len(ids) >= limit {
			break
		}
	}
	return ids, nil
}

func (s *Service) songBriefByID(ctx context.Context, id int64) (songBrief, error) {
	rows, err := s.queryBriefs(ctx, briefSelect+` WHERE s.id = ? LIMIT 1`, id)
	if err != nil {
		return songBrief{}, err
	}
	if len(rows) == 0 {
		return songBrief{}, fmt.Errorf("song %d not found", id)
	}
	return rows[0], nil
}
