package db

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Aunali321/korus/internal/models"
)

// Table aliases assumed by the column constants below: songs s, albums al,
// the album's credited artist ar, and a standalone artist a.
const (
	// SongColumns selects a song with its album and album artist.
	// duration_ms is exposed in seconds for API compatibility.
	SongColumns = `s.id, s.album_id, s.title, s.track_number, s.duration_ms / 1000 as duration, s.file_path,
		ar.id, ar.name, al.id, al.title, al.year, al.cover_path`

	// SongJoins resolves the album and album artist for SongColumns.
	SongJoins = `LEFT JOIN albums al ON al.id = s.album_id
		LEFT JOIN artists ar ON ar.id = al.artist_id`

	// AlbumColumns selects an album with its credited artist.
	AlbumColumns = `al.id, al.artist_id, al.title, al.year, al.cover_path, al.mbid, al.created_at,
		ar.id, ar.name`

	// AlbumJoins resolves the album artist for AlbumColumns.
	AlbumJoins = `LEFT JOIN artists ar ON ar.id = al.artist_id`

	ArtistColumns = `a.id, a.name, a.bio, a.image_path, a.mbid, a.external_id, a.created_at`
)

// ScanSong maps one SongColumns row. Song.Artists holds the credited
// performers and is filled separately by PopulateSongArtists.
func ScanSong(row interface{ Scan(...any) error }) (models.Song, error) {
	var song models.Song
	var track, duration, year sql.NullInt64
	var artistID, albumID sql.NullInt64
	var artistName, albumTitle, coverPath sql.NullString

	err := row.Scan(
		&song.ID, &song.AlbumID, &song.Title, &track, &duration, &song.FilePath,
		&artistID, &artistName, &albumID, &albumTitle, &year, &coverPath,
	)
	if err != nil {
		return song, err
	}

	if track.Valid {
		t := int(track.Int64)
		song.TrackNumber = &t
	}
	if duration.Valid {
		d := int(duration.Int64)
		song.Duration = &d
	}
	if albumID.Valid {
		album := &models.Album{ID: albumID.Int64, Title: albumTitle.String, CoverPath: coverPath.String}
		if year.Valid {
			y := int(year.Int64)
			album.Year = &y
		}
		if artistID.Valid {
			album.ArtistID = &artistID.Int64
			album.Artist = &models.Artist{ID: artistID.Int64, Name: artistName.String}
		}
		song.Album = album
	}

	return song, nil
}

// ScanSongs maps a full result set of SongColumns rows.
func ScanSongs(rows *sql.Rows) ([]models.Song, error) {
	songs := []models.Song{}
	for rows.Next() {
		song, err := ScanSong(rows)
		if err != nil {
			continue
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}

// ScanAlbum maps one AlbumColumns row.
func ScanAlbum(row interface{ Scan(...any) error }) (models.Album, error) {
	var al models.Album
	var albumArtistID, artistID, year sql.NullInt64
	var coverPath, mbid, artistName sql.NullString

	err := row.Scan(&al.ID, &albumArtistID, &al.Title, &year, &coverPath, &mbid, &al.CreatedAt,
		&artistID, &artistName)
	if err != nil {
		return al, err
	}

	al.CoverPath = coverPath.String
	if albumArtistID.Valid {
		al.ArtistID = &albumArtistID.Int64
	}
	if year.Valid {
		y := int(year.Int64)
		al.Year = &y
	}
	if mbid.Valid {
		al.MBID = &mbid.String
	}
	// Compilations have no album artist; the frontend renders that as "Various".
	if artistID.Valid {
		al.Artist = &models.Artist{ID: artistID.Int64, Name: artistName.String}
	}
	return al, nil
}

// ScanAlbums maps a full result set of AlbumColumns rows.
func ScanAlbums(rows *sql.Rows) ([]models.Album, error) {
	albums := []models.Album{}
	for rows.Next() {
		al, err := ScanAlbum(rows)
		if err != nil {
			continue
		}
		albums = append(albums, al)
	}
	return albums, rows.Err()
}

// ScanArtist maps one ArtistColumns row.
func ScanArtist(row interface{ Scan(...any) error }) (models.Artist, error) {
	var a models.Artist
	var bio, imagePath, mbid, externalID sql.NullString
	if err := row.Scan(&a.ID, &a.Name, &bio, &imagePath, &mbid, &externalID, &a.CreatedAt); err != nil {
		return a, err
	}
	a.Bio = bio.String
	a.ImagePath = imagePath.String
	if mbid.Valid {
		a.MBID = &mbid.String
	}
	if externalID.Valid {
		a.ExternalID = &externalID.String
	}
	return a, nil
}

// ScanArtists maps a full result set of ArtistColumns rows.
func ScanArtists(rows *sql.Rows) ([]models.Artist, error) {
	artists := []models.Artist{}
	for rows.Next() {
		a, err := ScanArtist(rows)
		if err != nil {
			continue
		}
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

// ChunkSize bounds an IN clause. SQLite rejects any statement with more than
// 32766 bound variables, which a long playlist would otherwise reach.
const ChunkSize = 500

// Placeholders builds "?, ?, ?" for an IN clause of n values.
func Placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

// Args converts ids to the []any a variadic query call needs.
func Args(ids []int64) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

// Chunks splits ids into batches small enough to bind in one statement.
func Chunks(ids []int64) [][]int64 {
	batches := [][]int64{}
	for start := 0; start < len(ids); start += ChunkSize {
		end := min(start+ChunkSize, len(ids))
		batches = append(batches, ids[start:end])
	}
	return batches
}

// PopulateSongArtists fills each song's credited performers from song_artists
// in one query, preserving credit order within each song.
func PopulateSongArtists(ctx context.Context, db *sql.DB, songs []models.Song) error {
	if len(songs) == 0 {
		return nil
	}
	ids := make([]int64, len(songs))
	index := make(map[int64]*models.Song, len(songs))
	for i := range songs {
		ids[i] = songs[i].ID
		index[songs[i].ID] = &songs[i]
	}

	for _, batch := range Chunks(ids) {
		if err := populateBatch(ctx, db, batch, index); err != nil {
			return err
		}
	}
	return nil
}

func populateBatch(ctx context.Context, db *sql.DB, ids []int64, index map[int64]*models.Song) error {
	rows, err := db.QueryContext(ctx, `
		SELECT sa.song_id, `+ArtistColumns+`
		FROM song_artists sa
		JOIN artists a ON a.id = sa.artist_id
		WHERE sa.song_id IN (`+Placeholders(len(ids))+`)
		ORDER BY sa.song_id, sa.position
	`, Args(ids)...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var songID int64
		var a models.Artist
		var bio, imagePath, mbid, externalID sql.NullString
		if err := rows.Scan(&songID, &a.ID, &a.Name, &bio, &imagePath, &mbid, &externalID, &a.CreatedAt); err != nil {
			continue
		}
		a.Bio = bio.String
		a.ImagePath = imagePath.String
		if mbid.Valid {
			a.MBID = &mbid.String
		}
		if externalID.Valid {
			a.ExternalID = &externalID.String
		}
		if song, ok := index[songID]; ok {
			song.Artists = append(song.Artists, a)
		}
	}
	return rows.Err()
}

// ArtistsForSong returns one song's credited performers in credit order.
func ArtistsForSong(ctx context.Context, db *sql.DB, songID int64) ([]models.Artist, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+ArtistColumns+`
		FROM song_artists sa
		JOIN artists a ON a.id = sa.artist_id
		WHERE sa.song_id = ?
		ORDER BY sa.position
	`, songID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return ScanArtists(rows)
}
