package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"korus/internal/config"
	"korus/internal/database"
	"korus/internal/models"
)

const (
	defaultPerArtistCap          = 3
	userSeedHalfLife             = 48 * time.Hour
	userSeedMinimumSources       = 5
	userRecentPlayLimit          = 200
	userLikedSongLimit           = 200
	radioscanCandidateMultiplier = 6
	recommendCandidateMultiplier = 6
)

type embeddingRecord struct {
	SongID   int
	Vector   []float32
	Dim      int
	Segments int
	Method   string
	Model    string
	Updated  time.Time
}

type candidateScore struct {
	SongID int
	Score  float64
}

type RecommenderService struct {
	db      *database.DB
	cfg     *config.RecommenderConfig
	mu      sync.RWMutex
	vectors map[int]*embeddingRecord
	order   []int
	rng     *rand.Rand
}

func NewRecommenderService(db *database.DB, cfg *config.RecommenderConfig) (*RecommenderService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("recommender config is nil")
	}

	service := &RecommenderService{
		db:      db,
		cfg:     cfg,
		vectors: make(map[int]*embeddingRecord),
		order:   make([]int, 0),
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if !cfg.Enabled {
		return service, nil
	}

	if err := service.LoadIndex(context.Background()); err != nil {
		return nil, err
	}

	return service, nil
}

func (s *RecommenderService) Enabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Enabled
}

func (s *RecommenderService) LoadIndex(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}

	query := `
		SELECT song_id, embedding, dim, segments, method, model, updated_at
		FROM song_embeddings
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to load embeddings: %w", err)
	}
	defer rows.Close()

	newVectors := make(map[int]*embeddingRecord)
	newOrder := make([]int, 0)

	for rows.Next() {
		var songID int
		var vectorArray pgtype.Array[float32]
		var dim pgtype.Int2
		var segments pgtype.Int2
		var method pgtype.Text
		var model pgtype.Text
		var updated time.Time

		if err := rows.Scan(
			&songID,
			&vectorArray,
			&dim,
			&segments,
			&method,
			&model,
			&updated,
		); err != nil {
			return fmt.Errorf("failed to scan embedding row: %w", err)
		}

		if !vectorArray.Valid {
			continue
		}

		rawVector := append([]float32(nil), vectorArray.Elements...)

		record := s.buildEmbeddingRecord(songID, rawVector, int(dim.Int16), int(segments.Int16), method.String, model.String, updated)
		if record == nil {
			continue
		}

		newVectors[songID] = record
		newOrder = append(newOrder, songID)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.vectors = newVectors
	s.order = newOrder
	s.mu.Unlock()

	return nil
}

func (s *RecommenderService) RefreshSongs(ctx context.Context, songIDs []int) error {
	if !s.Enabled() || len(songIDs) == 0 {
		return nil
	}

	records, err := s.fetchEmbeddings(ctx, songIDs)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	orderSet := make(map[int]struct{}, len(s.order))
	for _, id := range s.order {
		orderSet[id] = struct{}{}
	}

	for _, songID := range songIDs {
		record, ok := records[songID]
		if ok {
			s.vectors[songID] = record
			if _, exists := orderSet[songID]; !exists {
				s.order = append(s.order, songID)
			}
		} else {
			delete(s.vectors, songID)
			s.removeFromOrderLocked(songID)
		}
	}

	return nil
}

func (s *RecommenderService) SimilarSongs(ctx context.Context, songID int, limit int) ([]models.Song, error) {
	if !s.Enabled() {
		return nil, errors.New("recommender disabled")
	}

	if limit <= 0 || limit > s.cfg.SimilarityLimit {
		limit = s.cfg.SimilarityLimit
	}

	s.mu.RLock()
	base, ok := s.vectors[songID]
	if !ok {
		s.mu.RUnlock()
		_ = s.enqueueEmbeddingJob(ctx, songID)
		return s.fallbackByMetadata(ctx, songID, limit)
	}

	candidates := make([]candidateScore, 0, len(s.vectors))
	for id, record := range s.vectors {
		if id == songID {
			continue
		}
		score := computeSimilarity(base, record)
		candidates = append(candidates, candidateScore{SongID: id, Score: score})
	}
	s.mu.RUnlock()

	if len(candidates) == 0 {
		return s.fallbackByMetadata(ctx, songID, limit)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	maxCandidates := limit * recommendCandidateMultiplier
	if maxCandidates > len(candidates) {
		maxCandidates = len(candidates)
	}

	topIDs := make([]int, maxCandidates)
	for i := 0; i < maxCandidates; i++ {
		topIDs[i] = candidates[i].SongID
	}

	songMap, err := s.fetchSongs(ctx, topIDs)
	if err != nil {
		return nil, err
	}

	artistCounts := make(map[int]int)
	filtered := make([]models.Song, 0, limit)

	for _, candidate := range candidates {
		song, exists := songMap[candidate.SongID]
		if !exists {
			continue
		}
		artistID := 0
		if song.ArtistID != nil {
			artistID = *song.ArtistID
		}
		if artistID != 0 && artistCounts[artistID] >= defaultPerArtistCap {
			continue
		}
		artistCounts[artistID]++
		filtered = append(filtered, song)
		if len(filtered) >= limit {
			break
		}
	}

	if len(filtered) == 0 {
		return s.fallbackByMetadata(ctx, songID, limit)
	}

	return filtered, nil
}

func (s *RecommenderService) UserRecommendations(ctx context.Context, userID int, limit int) ([]models.Song, error) {
	if !s.Enabled() {
		return nil, errors.New("recommender disabled")
	}

	if limit <= 0 || limit > s.cfg.SimilarityLimit {
		limit = s.cfg.SimilarityLimit
	}

	seedRecord, listenedSet, totalSources, err := s.buildUserSeed(ctx, userID)
	if err != nil {
		return nil, err
	}

	if seedRecord == nil || totalSources < userSeedMinimumSources {
		if s.cfg.ColdStartPolicy == "random_diverse" {
			return s.randomDiverse(ctx, limit)
		}
		return []models.Song{}, nil
	}

	s.mu.RLock()
	allCandidates := make([]candidateScore, 0, len(s.vectors))
	for id, record := range s.vectors {
		score := computeSimilarity(seedRecord, record)
		allCandidates = append(allCandidates, candidateScore{SongID: id, Score: score})
	}
	s.mu.RUnlock()

	if len(allCandidates) == 0 {
		return s.randomDiverse(ctx, limit)
	}

	sort.Slice(allCandidates, func(i, j int) bool {
		return allCandidates[i].Score > allCandidates[j].Score
	})

	maxCandidates := limit * recommendCandidateMultiplier
	if maxCandidates > len(allCandidates) {
		maxCandidates = len(allCandidates)
	}

	topIDs := make([]int, maxCandidates)
	for i := 0; i < maxCandidates; i++ {
		topIDs[i] = allCandidates[i].SongID
	}

	songMap, err := s.fetchSongs(ctx, topIDs)
	if err != nil {
		return nil, err
	}

	artistCounts := make(map[int]int)
	results := make([]models.Song, 0, limit)

	for _, candidate := range allCandidates {
		song, exists := songMap[candidate.SongID]
		if !exists {
			continue
		}

		artistID := 0
		if song.ArtistID != nil {
			artistID = *song.ArtistID
		}
		if artistID != 0 && artistCounts[artistID] >= defaultPerArtistCap {
			continue
		}

		score := candidate.Score
		if _, played := listenedSet[candidate.SongID]; played {
			score -= 0.05
		}

		recencyBoost := computeRecencyBoost(song.DateAdded)
		jitter := (s.rng.Float64() - 0.5) * 0.01
		adjusted := score + recencyBoost + jitter
		if adjusted <= 0 {
			continue
		}

		artistCounts[artistID]++
		results = append(results, song)
		if len(results) >= limit {
			break
		}
	}

	if len(results) == 0 {
		return s.randomDiverse(ctx, limit)
	}

	return results, nil
}

func (s *RecommenderService) Radio(ctx context.Context, seedSongIDs []int, limit int) ([]models.Song, error) {
	if !s.Enabled() {
		return nil, errors.New("recommender disabled")
	}

	if limit <= 0 || limit > s.cfg.SimilarityLimit {
		limit = s.cfg.SimilarityLimit
	}

	seedRecord, _, _, err := s.buildSeedFromSongs(ctx, seedSongIDs)
	if err != nil {
		return nil, err
	}

	if seedRecord == nil {
		if s.cfg.ColdStartPolicy == "random_diverse" {
			return s.randomDiverse(ctx, limit)
		}
		return []models.Song{}, nil
	}

	seedSet := make(map[int]struct{}, len(seedSongIDs))
	for _, id := range seedSongIDs {
		seedSet[id] = struct{}{}
	}

	s.mu.RLock()
	allCandidates := make([]candidateScore, 0, len(s.vectors))
	for id, record := range s.vectors {
		allCandidates = append(allCandidates, candidateScore{SongID: id, Score: computeSimilarity(seedRecord, record)})
	}
	s.mu.RUnlock()

	sort.Slice(allCandidates, func(i, j int) bool {
		return allCandidates[i].Score > allCandidates[j].Score
	})

	maxCandidates := radioscanCandidateMultiplier * limit
	if maxCandidates > len(allCandidates) {
		maxCandidates = len(allCandidates)
	}

	topIDs := make([]int, maxCandidates)
	for i := 0; i < maxCandidates; i++ {
		topIDs[i] = allCandidates[i].SongID
	}

	songMap, err := s.fetchSongs(ctx, topIDs)
	if err != nil {
		return nil, err
	}

	artistCounts := make(map[int]int)
	selected := make([]models.Song, 0, limit)
	selectedRecords := make([]*embeddingRecord, 0, limit)

	for _, candidate := range allCandidates {
		if len(selected) >= limit {
			break
		}

		if _, seeded := seedSet[candidate.SongID]; seeded {
			continue
		}

		song, exists := songMap[candidate.SongID]
		if !exists {
			continue
		}

		record := s.getEmbedding(candidate.SongID)
		if record == nil {
			continue
		}

		artistID := 0
		if song.ArtistID != nil {
			artistID = *song.ArtistID
		}
		if artistID != 0 && artistCounts[artistID] >= defaultPerArtistCap {
			continue
		}

		penalty := 0.0
		for _, sel := range selectedRecords {
			penalty = math.Max(penalty, computeSimilarity(record, sel))
		}

		adjusted := candidate.Score - penalty*0.05 + computeRecencyBoost(song.DateAdded)
		if adjusted <= 0 {
			continue
		}

		artistCounts[artistID]++
		selected = append(selected, song)
		selectedRecords = append(selectedRecords, record)
	}

	if len(selected) == 0 {
		return s.randomDiverse(ctx, limit)
	}

	return selected, nil
}

func (s *RecommenderService) randomDiverse(ctx context.Context, limit int) ([]models.Song, error) {
	s.mu.RLock()
	if len(s.order) == 0 {
		s.mu.RUnlock()
		return []models.Song{}, nil
	}

	temp := make([]int, len(s.order))
	copy(temp, s.order)
	s.mu.RUnlock()

	s.rng.Shuffle(len(temp), func(i, j int) {
		temp[i], temp[j] = temp[j], temp[i]
	})

	maxCandidates := limit * recommendCandidateMultiplier
	if maxCandidates > len(temp) {
		maxCandidates = len(temp)
	}

	sample := temp[:maxCandidates]
	songMap, err := s.fetchSongs(ctx, sample)
	if err != nil {
		return nil, err
	}

	artistCounts := make(map[int]int)
	results := make([]models.Song, 0, limit)

	for _, id := range sample {
		song, exists := songMap[id]
		if !exists {
			continue
		}
		artistID := 0
		if song.ArtistID != nil {
			artistID = *song.ArtistID
		}
		if artistID != 0 && artistCounts[artistID] >= defaultPerArtistCap {
			continue
		}
		artistCounts[artistID]++
		results = append(results, song)
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

func (s *RecommenderService) buildUserSeed(ctx context.Context, userID int) (*embeddingRecord, map[int]struct{}, int, error) {
	playsQuery := `
		SELECT song_id, played_at
		FROM play_history
		WHERE user_id = $1
		ORDER BY played_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, playsQuery, userID, userRecentPlayLimit)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to load play history: %w", err)
	}
	defer rows.Close()

	listened := make(map[int]struct{})
	weights := make(map[int]float64)
	maxDim := 0
	maxSegments := 0

	for rows.Next() {
		var songID int
		var playedAt time.Time
		if err := rows.Scan(&songID, &playedAt); err != nil {
			continue
		}
		listened[songID] = struct{}{}
		age := time.Since(playedAt)
		weight := math.Exp(-age.Seconds() * math.Ln2 / userSeedHalfLife.Seconds())
		weights[songID] += weight

		rec := s.getEmbedding(songID)
		if rec != nil {
			if rec.Dim > maxDim {
				maxDim = rec.Dim
			}
			if rec.Segments > maxSegments {
				maxSegments = rec.Segments
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, 0, err
	}

	likesQuery := `
		SELECT song_id, liked_at
		FROM liked_songs
		WHERE user_id = $1
		ORDER BY liked_at DESC
		LIMIT $2
	`

	likeRows, err := s.db.QueryContext(ctx, likesQuery, userID, userLikedSongLimit)
	if err != nil {
		return nil, listened, len(weights), fmt.Errorf("failed to load liked songs: %w", err)
	}
	defer likeRows.Close()

	for likeRows.Next() {
		var songID int
		var likedAt time.Time
		if err := likeRows.Scan(&songID, &likedAt); err != nil {
			continue
		}
		weights[songID] += 1.5

		rec := s.getEmbedding(songID)
		if rec != nil {
			if rec.Dim > maxDim {
				maxDim = rec.Dim
			}
			if rec.Segments > maxSegments {
				maxSegments = rec.Segments
			}
		}
	}

	if err := likeRows.Err(); err != nil {
		return nil, listened, len(weights), err
	}

	if len(weights) == 0 || maxDim == 0 {
		return nil, listened, len(weights), nil
	}

	vectorLength := (maxSegments + 1) * maxDim
	accumulator := make([]float64, vectorLength)
	var totalWeight float64

	for songID, weight := range weights {
		rec := s.getEmbedding(songID)
		if rec == nil {
			_ = s.enqueueEmbeddingJob(ctx, songID)
			continue
		}

		totalWeight += weight

		for i := 0; i < vectorLength && i < len(rec.Vector); i++ {
			accumulator[i] += float64(rec.Vector[i]) * weight
		}
	}

	if totalWeight == 0 {
		return nil, listened, len(weights), nil
	}

	combined := make([]float32, vectorLength)
	for i := range accumulator {
		combined[i] = float32(accumulator[i] / totalWeight)
	}

	normalizeBlocks(combined, maxDim)

	seed := &embeddingRecord{
		SongID:   0,
		Vector:   combined,
		Dim:      maxDim,
		Segments: maxSegments,
	}

	return seed, listened, len(weights), nil
}

func (s *RecommenderService) buildSeedFromSongs(ctx context.Context, songIDs []int) (*embeddingRecord, map[int]struct{}, int, error) {
	if len(songIDs) == 0 {
		return nil, nil, 0, nil
	}

	weights := make(map[int]float64)
	for _, id := range songIDs {
		weights[id] = weights[id] + 1
	}

	maxDim := 0
	maxSegments := 0

	for songID := range weights {
		rec := s.getEmbedding(songID)
		if rec == nil {
			_ = s.enqueueEmbeddingJob(ctx, songID)
			continue
		}
		if rec.Dim > maxDim {
			maxDim = rec.Dim
		}
		if rec.Segments > maxSegments {
			maxSegments = rec.Segments
		}
	}

	if maxDim == 0 {
		return nil, nil, 0, nil
	}

	vectorLength := (maxSegments + 1) * maxDim
	accumulator := make([]float64, vectorLength)
	var totalWeight float64

	for songID, weight := range weights {
		rec := s.getEmbedding(songID)
		if rec == nil {
			continue
		}
		totalWeight += weight
		for i := 0; i < vectorLength && i < len(rec.Vector); i++ {
			accumulator[i] += float64(rec.Vector[i]) * weight
		}
	}

	if totalWeight == 0 {
		return nil, nil, 0, nil
	}

	combined := make([]float32, vectorLength)
	for i := range accumulator {
		combined[i] = float32(accumulator[i] / totalWeight)
	}

	normalizeBlocks(combined, maxDim)

	seed := &embeddingRecord{
		SongID:   0,
		Vector:   combined,
		Dim:      maxDim,
		Segments: maxSegments,
	}

	seedSet := make(map[int]struct{}, len(songIDs))
	for _, id := range songIDs {
		seedSet[id] = struct{}{}
	}

	return seed, seedSet, len(weights), nil
}

func (s *RecommenderService) buildEmbeddingRecord(songID int, vector []float32, dim int, segments int, method string, model string, updated time.Time) *embeddingRecord {
	if dim <= 0 || len(vector) == 0 {
		return nil
	}

	if segments < 0 {
		segments = 0
	}

	if blockCount := len(vector) / dim; blockCount > 0 {
		actualSegments := blockCount - 1
		if actualSegments < 0 {
			actualSegments = 0
		}
		if segments == 0 || actualSegments < segments {
			segments = actualSegments
		}
	}

	return &embeddingRecord{
		SongID:   songID,
		Vector:   vector,
		Dim:      dim,
		Segments: segments,
		Method:   method,
		Model:    model,
		Updated:  updated,
	}
}

func (s *RecommenderService) fetchEmbeddings(ctx context.Context, songIDs []int) (map[int]*embeddingRecord, error) {
	if len(songIDs) == 0 {
		return map[int]*embeddingRecord{}, nil
	}

	query := `
		SELECT song_id, embedding, dim, segments, method, model, updated_at
		FROM song_embeddings
		WHERE song_id = ANY($1)
	`

	rows, err := s.db.QueryContext(ctx, query, songIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch embeddings: %w", err)
	}
	defer rows.Close()

	records := make(map[int]*embeddingRecord)
	for rows.Next() {
		var songID int
		var vectorArray pgtype.Array[float32]
		var dim pgtype.Int2
		var segments pgtype.Int2
		var method pgtype.Text
		var model pgtype.Text
		var updated time.Time

		if err := rows.Scan(
			&songID,
			&vectorArray,
			&dim,
			&segments,
			&method,
			&model,
			&updated,
		); err != nil {
			return nil, fmt.Errorf("failed to scan embedding row: %w", err)
		}

		if !vectorArray.Valid {
			continue
		}

		rawVector := append([]float32(nil), vectorArray.Elements...)

		record := s.buildEmbeddingRecord(songID, rawVector, int(dim.Int16), int(segments.Int16), method.String, model.String, updated)
		if record != nil {
			records[songID] = record
		}
	}

	return records, rows.Err()
}

func (s *RecommenderService) fetchSongs(ctx context.Context, songIDs []int) (map[int]models.Song, error) {
	if len(songIDs) == 0 {
		return map[int]models.Song{}, nil
	}

	query := `
		SELECT
			s.id,
			s.title,
			s.album_id,
			s.artist_id,
			s.track_number,
			s.disc_number,
			s.duration,
			s.file_path,
			s.file_size,
			s.file_modified,
			s.bitrate,
			s.format,
			s.cover_path,
			s.date_added,
			ar.id,
			ar.name,
			a.id,
			a.name
		FROM songs s
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE s.id = ANY($1)
	`

	rows, err := s.db.QueryContext(ctx, query, songIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch songs: %w", err)
	}
	defer rows.Close()

	result := make(map[int]models.Song)

	for rows.Next() {
		var song models.Song
		var albumIDVal pgtype.Int4
		var artistIDVal pgtype.Int4
		var trackNumberVal pgtype.Int4
		var discNumberVal pgtype.Int4
		var bitrateVal pgtype.Int4
		var formatVal pgtype.Text
		var coverPathVal pgtype.Text
		var artistJoinID pgtype.Int4
		var artistJoinName pgtype.Text
		var albumJoinID pgtype.Int4
		var albumJoinName pgtype.Text

		if err := rows.Scan(
			&song.ID,
			&song.Title,
			&albumIDVal,
			&artistIDVal,
			&trackNumberVal,
			&discNumberVal,
			&song.Duration,
			&song.FilePath,
			&song.FileSize,
			&song.FileModified,
			&bitrateVal,
			&formatVal,
			&coverPathVal,
			&song.DateAdded,
			&artistJoinID,
			&artistJoinName,
			&albumJoinID,
			&albumJoinName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan song row: %w", err)
		}

		if albumIDVal.Valid {
			id := int(albumIDVal.Int32)
			song.AlbumID = &id
		}
		if artistIDVal.Valid {
			id := int(artistIDVal.Int32)
			song.ArtistID = &id
		}

		if trackNumberVal.Valid {
			val := int(trackNumberVal.Int32)
			song.TrackNumber = &val
		}

		if discNumberVal.Valid {
			song.DiscNumber = int(discNumberVal.Int32)
		} else {
			song.DiscNumber = 1
		}

		if bitrateVal.Valid {
			val := int(bitrateVal.Int32)
			song.Bitrate = &val
		}

		if formatVal.Valid {
			val := formatVal.String
			song.Format = &val
		}

		if coverPathVal.Valid {
			val := coverPathVal.String
			song.CoverPath = &val
		}

		if artistJoinID.Valid && artistJoinName.Valid {
			id := int(artistJoinID.Int32)
			song.Artist = &models.Artist{ID: id, Name: artistJoinName.String}
		}
		if albumJoinID.Valid && albumJoinName.Valid {
			id := int(albumJoinID.Int32)
			song.Album = &models.Album{ID: id, Name: albumJoinName.String}
		}

		result[song.ID] = song
	}

	return result, rows.Err()
}

func (s *RecommenderService) fallbackByMetadata(ctx context.Context, songID int, limit int) ([]models.Song, error) {
	query := `
		SELECT album_id, artist_id
		FROM songs
		WHERE id = $1
	`

	var albumID pgtype.Int4
	var artistID pgtype.Int4
	if err := s.db.QueryRowContext(ctx, query, songID).Scan(&albumID, &artistID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []models.Song{}, nil
		}
		return nil, fmt.Errorf("failed to load song metadata: %w", err)
	}

	collected := make([]models.Song, 0, limit)
	seen := make(map[int]struct{})

	if albumID.Valid {
		albumSongsQuery := `
			SELECT id
			FROM songs
			WHERE album_id = $1 AND id <> $2
			ORDER BY disc_number, track_number, title
			LIMIT $3
		`

		albumRows, err := s.db.QueryContext(ctx, albumSongsQuery, albumID.Int32, songID, limit)
		if err == nil {
			ids := make([]int, 0)
			for albumRows.Next() {
				var id int
				if err := albumRows.Scan(&id); err == nil {
					ids = append(ids, id)
				}
			}
			albumRows.Close()
			songMap, _ := s.fetchSongs(ctx, ids)
			for _, id := range ids {
				if song, ok := songMap[id]; ok {
					collected = append(collected, song)
					seen[id] = struct{}{}
					if len(collected) >= limit {
						return collected, nil
					}
				}
			}
		}
	}

	if artistID.Valid {
		remaining := limit - len(collected)
		if remaining > 0 {
			artistSongsQuery := `
				SELECT id
				FROM songs
				WHERE artist_id = $1 AND id <> $2
				ORDER BY date_added DESC
				LIMIT $3
			`

			artistRows, err := s.db.QueryContext(ctx, artistSongsQuery, artistID.Int32, songID, remaining*2)
			if err == nil {
				ids := make([]int, 0)
				for artistRows.Next() {
					var id int
					if err := artistRows.Scan(&id); err == nil {
						if _, exists := seen[id]; exists {
							continue
						}
						ids = append(ids, id)
					}
				}
				artistRows.Close()
				songMap, _ := s.fetchSongs(ctx, ids)
				for _, id := range ids {
					if song, ok := songMap[id]; ok {
						collected = append(collected, song)
						if len(collected) >= limit {
							return collected, nil
						}
					}
				}
			}
		}
	}

	return collected, nil
}

func (s *RecommenderService) getEmbedding(songID int) *embeddingRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vectors[songID]
}

func (s *RecommenderService) removeFromOrderLocked(songID int) {
	for i, id := range s.order {
		if id == songID {
			s.order = append(s.order[:i], s.order[i+1:]...)
			return
		}
	}
}

func (s *RecommenderService) enqueueEmbeddingJob(ctx context.Context, songID int) error {
	if !s.Enabled() {
		return nil
	}

	var filePath string
	query := `SELECT file_path FROM songs WHERE id = $1`
	if err := s.db.QueryRowContext(ctx, query, songID).Scan(&filePath); err != nil {
		return err
	}

	checkQuery := `
		SELECT EXISTS(
			SELECT 1
			FROM job_queue
			WHERE job_type = 'embedding_extract'
				AND payload ->> 'file_path' = $1
				AND status IN ('pending', 'processing')
		)
	`

	var exists bool
	if err := s.db.QueryRowContext(ctx, checkQuery, filePath).Scan(&exists); err == nil && exists {
		return nil
	}

	payload := map[string]interface{}{
		"file_path": filePath,
		"song_id":   songID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO job_queue (job_type, payload, status, created_at)
		VALUES ('embedding_extract', $1::jsonb, 'pending', NOW())
	`

	_, err = s.db.ExecContext(ctx, insertQuery, data)
	return err
}

func computeSimilarity(a, b *embeddingRecord) float64 {
	if a == nil || b == nil || a.Dim <= 0 || b.Dim <= 0 {
		return 0
	}

	dim := minInt(a.Dim, b.Dim)
	if dim <= 0 {
		return 0
	}

	global := dotBlock(a.Vector, b.Vector, 0, 0, dim)

	segments := minInt(effectiveSegments(a), effectiveSegments(b))
	segmentScore := 0.0
	if segments > 0 {
		for i := 0; i < segments; i++ {
			aOffset := (i + 1) * a.Dim
			bOffset := (i + 1) * b.Dim
			segmentScore += dotBlock(a.Vector, b.Vector, aOffset, bOffset, dim)
		}
		segmentScore /= float64(segments)
	}

	return 0.7*global + 0.3*segmentScore
}

func dotBlock(a, b []float32, aOffset, bOffset, length int) float64 {
	maxA := aOffset + length
	maxB := bOffset + length
	if maxA > len(a) {
		maxA = len(a)
	}
	if maxB > len(b) {
		maxB = len(b)
	}
	limit := minInt(maxA-aOffset, maxB-bOffset)
	if limit <= 0 {
		return 0
	}

	var sum float64
	for i := 0; i < limit; i++ {
		sum += float64(a[aOffset+i]) * float64(b[bOffset+i])
	}
	return sum
}

func effectiveSegments(record *embeddingRecord) int {
	if record == nil || record.Dim <= 0 {
		return 0
	}
	blocks := len(record.Vector) / record.Dim
	if blocks <= 1 {
		return 0
	}
	if record.Segments > 0 && record.Segments < blocks-1 {
		return record.Segments
	}
	return blocks - 1
}

func normalizeBlocks(vec []float32, dim int) {
	if dim <= 0 {
		return
	}
	blocks := len(vec) / dim
	if blocks == 0 {
		return
	}

	for b := 0; b < blocks; b++ {
		start := b * dim
		end := start + dim
		if end > len(vec) {
			end = len(vec)
		}
		var norm float64
		for i := start; i < end; i++ {
			norm += float64(vec[i]) * float64(vec[i])
		}
		if norm == 0 {
			continue
		}
		scale := float32(1 / math.Sqrt(norm))
		for i := start; i < end; i++ {
			vec[i] *= scale
		}
	}
}

func computeRecencyBoost(dateAdded time.Time) float64 {
	if dateAdded.IsZero() {
		return 0
	}
	ageDays := time.Since(dateAdded).Hours() / 24
	if ageDays < 0 {
		ageDays = 0
	}
	return math.Exp(-ageDays/30.0) * 0.05
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
