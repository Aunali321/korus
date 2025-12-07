package search

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"korus/internal/config"
	"korus/internal/database"
	"korus/internal/models"
)

type SearchService struct {
	index  bleve.Index
	db     *database.DB
	config *config.LibraryConfig
}

type SearchDocument struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // "song", "album", "artist"
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`
	Year        int    `json:"year"`
	Duration    int    `json:"duration"`
	Format      string `json:"format"`
	FilePath    string `json:"file_path"`
}

type SearchOptions struct {
	Query  string
	Type   string // "song", "album", "artist", or empty for all
	Limit  int
	Offset int
}

func NewSearchService(db *database.DB, config *config.LibraryConfig) (*SearchService, error) {
	indexPath := filepath.Join(config.CacheDir, "search_index")

	// Try to open existing index
	index, err := bleve.Open(indexPath)
	if err != nil {
		// Create new index if it doesn't exist
		index, err = createSearchIndex(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create search index: %w", err)
		}
	}

	return &SearchService{
		index:  index,
		db:     db,
		config: config,
	}, nil
}

func createSearchIndex(indexPath string) (bleve.Index, error) {
	// Create index mapping
	mapping := bleve.NewIndexMapping()

	// Configure text field mapping for better search
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Analyzer = "standard"
	textFieldMapping.Store = true
	textFieldMapping.Index = true

	// Configure keyword field mapping for exact matches
	keywordFieldMapping := bleve.NewKeywordFieldMapping()
	keywordFieldMapping.Store = true
	keywordFieldMapping.Index = true

	// Configure numeric field mapping
	numericFieldMapping := bleve.NewNumericFieldMapping()
	numericFieldMapping.Store = true
	numericFieldMapping.Index = true

	// Create document mapping
	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("type", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("title", textFieldMapping)
	docMapping.AddFieldMappingsAt("artist", textFieldMapping)
	docMapping.AddFieldMappingsAt("album", textFieldMapping)
	docMapping.AddFieldMappingsAt("album_artist", textFieldMapping)
	docMapping.AddFieldMappingsAt("year", numericFieldMapping)
	docMapping.AddFieldMappingsAt("duration", numericFieldMapping)
	docMapping.AddFieldMappingsAt("format", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("file_path", keywordFieldMapping)

	mapping.DefaultMapping = docMapping

	// Create index
	return bleve.New(indexPath, mapping)
}

func (s *SearchService) Close() error {
	return s.index.Close()
}

func (s *SearchService) IndexSong(ctx context.Context, song *models.Song) error {
	doc := SearchDocument{
		ID:       fmt.Sprintf("song_%d", song.ID),
		Type:     "song",
		Title:    song.Title,
		Duration: song.Duration,
		FilePath: song.FilePath,
	}

	// Add format if available
	if song.Format != nil {
		doc.Format = *song.Format
	}

	// Get artist and album information
	if song.ArtistID != nil {
		artist, err := s.getArtist(ctx, *song.ArtistID)
		if err == nil {
			doc.Artist = artist.Name
		}
	}

	if song.AlbumID != nil {
		album, err := s.getAlbum(ctx, *song.AlbumID)
		if err == nil {
			doc.Album = album.Name
			if album.Year != nil {
				doc.Year = *album.Year
			}

			// Get album artist
			if album.AlbumArtistID != nil {
				albumArtist, err := s.getArtist(ctx, *album.AlbumArtistID)
				if err == nil {
					doc.AlbumArtist = albumArtist.Name
				}
			}
		}
	}

	return s.index.Index(doc.ID, doc)
}

func (s *SearchService) IndexAlbum(ctx context.Context, album *models.Album) error {
	doc := SearchDocument{
		ID:    fmt.Sprintf("album_%d", album.ID),
		Type:  "album",
		Title: album.Name,
	}

	if album.Year != nil {
		doc.Year = *album.Year
	}

	// Get artist information
	if album.ArtistID != nil {
		artist, err := s.getArtist(ctx, *album.ArtistID)
		if err == nil {
			doc.Artist = artist.Name
		}
	}

	// Get album artist information
	if album.AlbumArtistID != nil {
		albumArtist, err := s.getArtist(ctx, *album.AlbumArtistID)
		if err == nil {
			doc.AlbumArtist = albumArtist.Name
		}
	}

	return s.index.Index(doc.ID, doc)
}

func (s *SearchService) IndexArtist(ctx context.Context, artist *models.Artist) error {
	doc := SearchDocument{
		ID:     fmt.Sprintf("artist_%d", artist.ID),
		Type:   "artist",
		Title:  artist.Name,
		Artist: artist.Name,
	}

	return s.index.Index(doc.ID, doc)
}

func (s *SearchService) RemoveFromIndex(itemType string, id int) error {
	docID := fmt.Sprintf("%s_%d", itemType, id)
	return s.index.Delete(docID)
}

func (s *SearchService) Search(ctx context.Context, options SearchOptions) (*models.SearchResults, error) {
	// Build search query
	query, err := s.buildSearchQuery(options)
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	// Create search request
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = options.Limit
	searchRequest.From = options.Offset
	searchRequest.Fields = []string{"*"}

	// Add highlighting
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.Highlight.AddField("title")
	searchRequest.Highlight.AddField("artist")
	searchRequest.Highlight.AddField("album")

	// Execute search
	searchResult, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to models
	return s.convertSearchResults(ctx, searchResult)
}

func (s *SearchService) buildSearchQuery(options SearchOptions) (query.Query, error) {
	var queries []query.Query

	// Main text query
	if options.Query != "" {
		// Create a disjunction query across multiple fields
		titleQuery := bleve.NewMatchQuery(options.Query)
		titleQuery.SetField("title")
		titleQuery.SetBoost(2.0) // Boost title matches

		artistQuery := bleve.NewMatchQuery(options.Query)
		artistQuery.SetField("artist")
		artistQuery.SetBoost(1.5) // Boost artist matches

		albumQuery := bleve.NewMatchQuery(options.Query)
		albumQuery.SetField("album")

		albumArtistQuery := bleve.NewMatchQuery(options.Query)
		albumArtistQuery.SetField("album_artist")

		// Create disjunction (OR) query
		textQuery := bleve.NewDisjunctionQuery(titleQuery, artistQuery, albumQuery, albumArtistQuery)
		queries = append(queries, textQuery)
	}

	// Type filter
	if options.Type != "" {
		typeQuery := bleve.NewTermQuery(options.Type)
		typeQuery.SetField("type")
		queries = append(queries, typeQuery)
	}

	// Combine queries with conjunction (AND)
	if len(queries) == 0 {
		return bleve.NewMatchAllQuery(), nil
	} else if len(queries) == 1 {
		return queries[0], nil
	} else {
		return bleve.NewConjunctionQuery(queries...), nil
	}
}

func (s *SearchService) convertSearchResults(ctx context.Context, searchResult *bleve.SearchResult) (*models.SearchResults, error) {
	results := &models.SearchResults{
		Songs:   []models.Song{},
		Albums:  []models.Album{},
		Artists: []models.Artist{},
	}

	for _, hit := range searchResult.Hits {
		// Extract type and ID from document ID
		parts := strings.SplitN(hit.ID, "_", 2)
		if len(parts) != 2 {
			continue
		}

		itemType := parts[0]
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		// Fetch full object from database
		switch itemType {
		case "song":
			song, err := s.getSong(ctx, id)
			if err == nil {
				results.Songs = append(results.Songs, *song)
			}
		case "album":
			album, err := s.getAlbum(ctx, id)
			if err == nil {
				results.Albums = append(results.Albums, *album)
			}
		case "artist":
			artist, err := s.getArtist(ctx, id)
			if err == nil {
				results.Artists = append(results.Artists, *artist)
			}
		}
	}

	return results, nil
}

func (s *SearchService) RebuildIndex(ctx context.Context) error {
	// Clear existing index
	if err := s.clearIndex(); err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	// Index all songs
	if err := s.indexAllSongs(ctx); err != nil {
		return fmt.Errorf("failed to index songs: %w", err)
	}

	// Index all albums
	if err := s.indexAllAlbums(ctx); err != nil {
		return fmt.Errorf("failed to index albums: %w", err)
	}

	// Index all artists
	if err := s.indexAllArtists(ctx); err != nil {
		return fmt.Errorf("failed to index artists: %w", err)
	}

	return nil
}

func (s *SearchService) clearIndex() error {
	// Get all document IDs
	query := bleve.NewMatchAllQuery()
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = 10000 // Large number to get all documents
	searchRequest.Fields = []string{}

	searchResult, err := s.index.Search(searchRequest)
	if err != nil {
		return err
	}

	// Delete all documents
	for _, hit := range searchResult.Hits {
		if err := s.index.Delete(hit.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *SearchService) indexAllSongs(ctx context.Context) error {
	query := `
		SELECT id, title, album_id, artist_id, track_number, disc_number, duration, 
			   file_path, file_size, file_modified, bitrate, format, date_added
		FROM songs
		ORDER BY id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var song models.Song
		err := rows.Scan(&song.ID, &song.Title, &song.AlbumID, &song.ArtistID,
			&song.TrackNumber, &song.DiscNumber, &song.Duration,
			&song.FilePath, &song.FileSize, &song.FileModified,
			&song.Bitrate, &song.Format, &song.DateAdded)
		if err != nil {
			continue
		}

		if err := s.IndexSong(ctx, &song); err != nil {
			// Log error but continue
			fmt.Printf("Failed to index song %d: %v\n", song.ID, err)
		}
	}

	return rows.Err()
}

func (s *SearchService) indexAllAlbums(ctx context.Context) error {
	query := `
		SELECT id, name, artist_id, album_artist_id, year, musicbrainz_id, cover_path, date_added
		FROM albums
		ORDER BY id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var album models.Album
		err := rows.Scan(&album.ID, &album.Name, &album.ArtistID, &album.AlbumArtistID,
			&album.Year, &album.MusicBrainzID, &album.CoverPath, &album.DateAdded)
		if err != nil {
			continue
		}

		if err := s.IndexAlbum(ctx, &album); err != nil {
			// Log error but continue
			fmt.Printf("Failed to index album %d: %v\n", album.ID, err)
		}
	}

	return rows.Err()
}

func (s *SearchService) indexAllArtists(ctx context.Context) error {
	query := `
		SELECT id, name, sort_name, musicbrainz_id
		FROM artists
		ORDER BY id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var artist models.Artist
		err := rows.Scan(&artist.ID, &artist.Name, &artist.SortName, &artist.MusicBrainzID)
		if err != nil {
			continue
		}

		if err := s.IndexArtist(ctx, &artist); err != nil {
			// Log error but continue
			fmt.Printf("Failed to index artist %d: %v\n", artist.ID, err)
		}
	}

	return rows.Err()
}

// Helper methods to fetch data from database
func (s *SearchService) getSong(ctx context.Context, id int) (*models.Song, error) {
	query := `
		SELECT id, title, album_id, artist_id, track_number, disc_number, duration, 
			   file_path, file_size, file_modified, bitrate, format, date_added
		FROM songs
		WHERE id = $1
	`

	var song models.Song
	err := s.db.QueryRowContext(ctx, query, id).
		Scan(&song.ID, &song.Title, &song.AlbumID, &song.ArtistID,
			&song.TrackNumber, &song.DiscNumber, &song.Duration,
			&song.FilePath, &song.FileSize, &song.FileModified,
			&song.Bitrate, &song.Format, &song.DateAdded)

	return &song, err
}

func (s *SearchService) getAlbum(ctx context.Context, id int) (*models.Album, error) {
	query := `
		SELECT id, name, artist_id, album_artist_id, year, musicbrainz_id, cover_path, date_added
		FROM albums
		WHERE id = $1
	`

	var album models.Album
	err := s.db.QueryRowContext(ctx, query, id).
		Scan(&album.ID, &album.Name, &album.ArtistID, &album.AlbumArtistID,
			&album.Year, &album.MusicBrainzID, &album.CoverPath, &album.DateAdded)

	return &album, err
}

func (s *SearchService) getArtist(ctx context.Context, id int) (*models.Artist, error) {
	query := `
		SELECT id, name, sort_name, musicbrainz_id
		FROM artists
		WHERE id = $1
	`

	var artist models.Artist
	err := s.db.QueryRowContext(ctx, query, id).
		Scan(&artist.ID, &artist.Name, &artist.SortName, &artist.MusicBrainzID)

	return &artist, err
}
