package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/models"
)

type SearchService struct {
	db *sql.DB
}

func NewSearchService(database *sql.DB) *SearchService {
	return &SearchService{db: database}
}

// SearchKind names one searchable entity type.
type SearchKind string

const (
	KindSong     SearchKind = "song"
	KindAlbum    SearchKind = "album"
	KindArtist   SearchKind = "artist"
	KindPlaylist SearchKind = "playlist"
)

var AllSearchKinds = []SearchKind{KindSong, KindAlbum, KindArtist, KindPlaylist}

type SearchResult struct {
	Songs     []models.Song     `json:"songs"`
	Albums    []models.Album    `json:"albums"`
	Artists   []models.Artist   `json:"artists"`
	Playlists []models.Playlist `json:"playlists"`
}

// Search looks up each requested kind by name. Playlist results are scoped to
// what the user may see.
func (s *SearchService) Search(ctx context.Context, userID int64, query string, kinds []SearchKind, limit, offset int) (SearchResult, error) {
	res := SearchResult{
		Songs:     []models.Song{},
		Albums:    []models.Album{},
		Artists:   []models.Artist{},
		Playlists: []models.Playlist{},
	}
	terms := searchTerms(query)
	if len(terms) == 0 {
		return res, nil
	}

	want := make(map[SearchKind]bool, len(kinds))
	for _, k := range kinds {
		want[k] = true
	}

	var err error
	if want[KindSong] {
		if res.Songs, err = s.songs(ctx, terms, limit, offset); err != nil {
			return res, fmt.Errorf("search songs: %w", err)
		}
	}
	if want[KindArtist] {
		if res.Artists, err = s.artists(ctx, terms, limit, offset); err != nil {
			return res, fmt.Errorf("search artists: %w", err)
		}
	}
	if want[KindAlbum] {
		if res.Albums, err = s.albums(ctx, terms, limit, offset); err != nil {
			return res, fmt.Errorf("search albums: %w", err)
		}
	}
	if want[KindPlaylist] {
		if res.Playlists, err = s.playlists(ctx, userID, terms, limit, offset); err != nil {
			return res, fmt.Errorf("search playlists: %w", err)
		}
	}
	return res, nil
}

// songs matches every term first, then falls back to matching any of them
// ranked by relevance. A library's titles follow whatever convention its owner
// used and often pack artist, year and other context into one line, so an
// all-terms match alone drops results a human would call obvious. Returning
// nothing also reads to the AI agent as "the library has nothing like this",
// which sends it looking somewhere worse.
func (s *SearchService) songs(ctx context.Context, terms []string, limit, offset int) ([]models.Song, error) {
	songs, err := s.matchSongs(ctx, ftsQuery(terms, "AND"), limit, offset)
	if err != nil || len(songs) > 0 || len(terms) == 1 {
		return songs, err
	}
	return s.matchSongs(ctx, ftsQuery(terms, "OR"), limit, offset)
}

func (s *SearchService) matchSongs(ctx context.Context, match string, limit, offset int) ([]models.Song, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+db.SongColumns+`
		FROM songs_fts fts
		JOIN songs s ON s.id = fts.rowid
		`+db.SongJoins+`
		WHERE songs_fts MATCH ?
		ORDER BY bm25(songs_fts)
		LIMIT ? OFFSET ?
	`, match, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	songs, err := db.ScanSongs(rows)
	if err != nil {
		return nil, err
	}
	if err := db.PopulateSongArtists(ctx, s.db, songs); err != nil {
		return nil, err
	}
	return songs, nil
}

func (s *SearchService) artists(ctx context.Context, terms []string, limit, offset int) ([]models.Artist, error) {
	where, args := likeAll(terms, "a.name")
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+db.ArtistColumns+`
		FROM artists a
		WHERE `+where+`
		ORDER BY LENGTH(a.name)
		LIMIT ? OFFSET ?
	`, append(args, limit, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return db.ScanArtists(rows)
}

func (s *SearchService) albums(ctx context.Context, terms []string, limit, offset int) ([]models.Album, error) {
	where, args := likeAll(terms, "al.title")
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+db.AlbumColumns+`
		FROM albums al
		`+db.AlbumJoins+`
		WHERE `+where+`
		ORDER BY LENGTH(al.title)
		LIMIT ? OFFSET ?
	`, append(args, limit, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return db.ScanAlbums(rows)
}

func (s *SearchService) playlists(ctx context.Context, userID int64, terms []string, limit, offset int) ([]models.Playlist, error) {
	where, args := likeAll(terms, "p.name")
	args = append([]any{userID}, args...)
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+playlistColumns+`
		FROM playlists p
		JOIN users u ON u.id = p.user_id
		WHERE (p.public = 1 OR p.user_id = ?) AND `+where+`
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`, append(args, limit, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playlists := []models.Playlist{}
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			continue
		}
		playlists = append(playlists, p)
	}
	return playlists, rows.Err()
}

// searchTerms splits a query into terms, dropping punctuation. Users and tag
// editors write the same title with wildly different separators, and FTS5
// treats most of that punctuation as query syntax.
func searchTerms(query string) []string {
	fields := strings.FieldsFunc(query, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	terms := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			terms = append(terms, f)
		}
	}
	return terms
}

// ftsQuery builds an FTS5 expression from already-sanitised terms. Each term
// is quoted so it is read as a literal, and given a prefix match so partial
// words still hit.
func ftsQuery(terms []string, op string) string {
	quoted := make([]string, len(terms))
	for i, t := range terms {
		quoted[i] = `"` + t + `"*`
	}
	return strings.Join(quoted, " "+op+" ")
}

// likeAll requires every term to appear in the column.
func likeAll(terms []string, column string) (string, []any) {
	clauses := make([]string, len(terms))
	args := make([]any, len(terms))
	for i, t := range terms {
		clauses[i] = column + " LIKE ?"
		args[i] = "%" + t + "%"
	}
	return strings.Join(clauses, " AND "), args
}
