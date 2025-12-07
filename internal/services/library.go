package services

import (
	"context"
	"fmt"

	"korus/internal/database"
	"korus/internal/models"
)

type LibraryService struct {
	db *database.DB
}

func NewLibraryService(db *database.DB) *LibraryService {
	return &LibraryService{db: db}
}

func (ls *LibraryService) GetStats(ctx context.Context) (*models.LibraryStats, error) {
	stats := &models.LibraryStats{}

	err := ls.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM songs").Scan(&stats.TotalSongs)
	if err != nil {
		return nil, fmt.Errorf("failed to get song count: %w", err)
	}

	err = ls.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM artists").Scan(&stats.TotalArtists)
	if err != nil {
		return nil, fmt.Errorf("failed to get artist count: %w", err)
	}

	err = ls.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM albums").Scan(&stats.TotalAlbums)
	if err != nil {
		return nil, fmt.Errorf("failed to get album count: %w", err)
	}

	err = ls.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(duration), 0) FROM songs").Scan(&stats.TotalDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get total duration: %w", err)
	}

	return stats, nil
}

func (ls *LibraryService) GetArtists(ctx context.Context, limit, offset int, sort string) ([]models.Artist, error) {
	query := `
		SELECT a.id, a.name, a.sort_name, a.musicbrainz_id,
			   COUNT(DISTINCT al.id) as album_count,
			   COUNT(DISTINCT s.id) as song_count
		FROM artists a
		LEFT JOIN albums al ON a.id = al.artist_id OR a.id = al.album_artist_id
		LEFT JOIN songs s ON a.id = s.artist_id
		GROUP BY a.id, a.name, a.sort_name, a.musicbrainz_id
	`

	switch sort {
	case "name":
		query += " ORDER BY a.name"
	case "name_desc":
		query += " ORDER BY a.name DESC"
	case "albums":
		query += " ORDER BY album_count DESC"
	case "songs":
		query += " ORDER BY song_count DESC"
	default:
		query += " ORDER BY a.sort_name, a.name"
	}

	query += " LIMIT $1 OFFSET $2"

	rows, err := ls.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query artists: %w", err)
	}
	defer rows.Close()

	artists := make([]models.Artist, 0)
	for rows.Next() {
		var artist models.Artist
		err := rows.Scan(&artist.ID, &artist.Name, &artist.SortName, &artist.MusicBrainzID,
			&artist.AlbumCount, &artist.SongCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artist: %w", err)
		}
		artists = append(artists, artist)
	}

	return artists, rows.Err()
}

func (ls *LibraryService) GetArtist(ctx context.Context, id int) (*models.Artist, error) {
	query := `
		SELECT a.id, a.name, a.sort_name, a.musicbrainz_id,
			   COUNT(DISTINCT al.id) as album_count,
			   COUNT(DISTINCT s.id) as song_count
		FROM artists a
		LEFT JOIN albums al ON a.id = al.artist_id OR a.id = al.album_artist_id
		LEFT JOIN songs s ON a.id = s.artist_id
		WHERE a.id = $1
		GROUP BY a.id, a.name, a.sort_name, a.musicbrainz_id
	`

	var artist models.Artist
	err := ls.db.QueryRowContext(ctx, query, id).
		Scan(&artist.ID, &artist.Name, &artist.SortName, &artist.MusicBrainzID,
			&artist.AlbumCount, &artist.SongCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get artist: %w", err)
	}

	albums, err := ls.GetArtistAlbums(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get artist albums: %w", err)
	}
	artist.Albums = albums

	topTracks, err := ls.GetArtistTopTracks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get artist top tracks: %w", err)
	}
	artist.TopTracks = topTracks

	return &artist, nil
}

func (ls *LibraryService) GetAlbums(ctx context.Context, limit, offset int, sort string, year *int) ([]models.Album, error) {
	query := `
		SELECT a.id, a.name, a.artist_id, a.album_artist_id, a.year, 
			   a.musicbrainz_id, a.cover_path, a.date_added,
			   ar.name as artist_name,
			   COUNT(DISTINCT s.id) as song_count,
			   COALESCE(SUM(s.duration), 0) as duration
		FROM albums a
		LEFT JOIN artists ar ON a.artist_id = ar.id
		LEFT JOIN songs s ON a.id = s.album_id
	`

	args := []interface{}{}
	argCount := 0

	if year != nil {
		query += " WHERE a.year = $1"
		args = append(args, *year)
		argCount++
	}

	query += " GROUP BY a.id, a.name, a.artist_id, a.album_artist_id, a.year, a.musicbrainz_id, a.cover_path, a.date_added, ar.name"

	switch sort {
	case "name":
		query += " ORDER BY a.name"
	case "name_desc":
		query += " ORDER BY a.name DESC"
	case "year":
		query += " ORDER BY a.year, a.name"
	case "year_desc":
		query += " ORDER BY a.year DESC, a.name"
	case "artist":
		query += " ORDER BY ar.name, a.name"
	case "date_added":
		query += " ORDER BY a.date_added DESC"
	default:
		query += " ORDER BY ar.name, a.year, a.name"
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount+1, argCount+2)
	args = append(args, limit, offset)

	rows, err := ls.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query albums: %w", err)
	}
	defer rows.Close()

	albums := make([]models.Album, 0)
	for rows.Next() {
		var album models.Album
		var artistName *string
		err := rows.Scan(&album.ID, &album.Name, &album.ArtistID, &album.AlbumArtistID,
			&album.Year, &album.MusicBrainzID, &album.CoverPath, &album.DateAdded,
			&artistName, &album.SongCount, &album.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed to scan album: %w", err)
		}

		if album.ArtistID != nil && artistName != nil {
			album.Artist = &models.Artist{
				ID:   *album.ArtistID,
				Name: *artistName,
			}
		}

		albums = append(albums, album)
	}

	return albums, rows.Err()
}

func (ls *LibraryService) GetAlbum(ctx context.Context, id int) (*models.Album, error) {
	query := `
		SELECT a.id, a.name, a.artist_id, a.album_artist_id, a.year, 
			   a.musicbrainz_id, a.cover_path, a.date_added,
			   ar.name as artist_name,
			   aar.name as album_artist_name,
			   COUNT(DISTINCT s.id) as song_count,
			   COALESCE(SUM(s.duration), 0) as duration
		FROM albums a
		LEFT JOIN artists ar ON a.artist_id = ar.id
		LEFT JOIN artists aar ON a.album_artist_id = aar.id
		LEFT JOIN songs s ON a.id = s.album_id
		WHERE a.id = $1
		GROUP BY a.id, a.name, a.artist_id, a.album_artist_id, a.year, 
				 a.musicbrainz_id, a.cover_path, a.date_added, ar.name, aar.name
	`

	var album models.Album
	var artistName, albumArtistName *string
	err := ls.db.QueryRowContext(ctx, query, id).
		Scan(&album.ID, &album.Name, &album.ArtistID, &album.AlbumArtistID,
			&album.Year, &album.MusicBrainzID, &album.CoverPath, &album.DateAdded,
			&artistName, &albumArtistName, &album.SongCount, &album.Duration)
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}

	if album.ArtistID != nil && artistName != nil {
		album.Artist = &models.Artist{
			ID:   *album.ArtistID,
			Name: *artistName,
		}
	}

	if album.AlbumArtistID != nil && albumArtistName != nil {
		album.AlbumArtist = &models.Artist{
			ID:   *album.AlbumArtistID,
			Name: *albumArtistName,
		}
	}

	songs, err := ls.GetAlbumSongs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get album songs: %w", err)
	}
	album.Songs = songs

	return &album, nil
}

func (ls *LibraryService) GetAlbumSongs(ctx context.Context, albumID int) ([]models.Song, error) {
	query := `
		SELECT s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified, 
			   s.bitrate, s.format, s.cover_path, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name
		FROM songs s
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE s.album_id = $1
		ORDER BY s.disc_number, s.track_number, s.title
	`

	return ls.querySongs(ctx, query, albumID)
}

func (ls *LibraryService) GetSongs(ctx context.Context, ids []int) ([]models.Song, error) {
	if len(ids) == 0 {
		return []models.Song{}, nil
	}

	query := `
		SELECT s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified, 
			   s.bitrate, s.format, s.cover_path, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name
		FROM songs s
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE s.id = ANY($1)
		ORDER BY s.title
	`

	return ls.querySongs(ctx, query, ids)
}

func (ls *LibraryService) GetSong(ctx context.Context, id int) (*models.Song, error) {
	query := `
		SELECT s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified, 
			   s.bitrate, s.format, s.cover_path, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name
		FROM songs s
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE s.id = $1
	`

	songs, err := ls.querySongs(ctx, query, id)
	if err != nil {
		return nil, err
	}

	if len(songs) == 0 {
		return nil, fmt.Errorf("song not found")
	}

	return &songs[0], nil
}

func (ls *LibraryService) querySongs(ctx context.Context, query string, args ...interface{}) ([]models.Song, error) {
	rows, err := ls.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query songs: %w", err)
	}
	defer rows.Close()

	songs := make([]models.Song, 0)
	for rows.Next() {
		var song models.Song
		var artistName, albumName *string
		err := rows.Scan(&song.ID, &song.Title, &song.AlbumID, &song.ArtistID,
			&song.TrackNumber, &song.DiscNumber, &song.Duration,
			&song.FilePath, &song.FileSize, &song.FileModified,
			&song.Bitrate, &song.Format, &song.CoverPath, &song.DateAdded,
			&artistName, &albumName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan song: %w", err)
		}

		// Set artist info
		if song.ArtistID != nil && artistName != nil {
			song.Artist = &models.Artist{
				ID:   *song.ArtistID,
				Name: *artistName,
			}
		}

		// Set album info
		if song.AlbumID != nil && albumName != nil {
			song.Album = &models.Album{
				ID:   *song.AlbumID,
				Name: *albumName,
			}
		}

		songs = append(songs, song)
	}

	// Load lyrics for all songs
	if err := ls.loadSongsLyrics(ctx, songs); err != nil {
		return nil, fmt.Errorf("failed to load lyrics: %w", err)
	}

	return songs, rows.Err()
}

func (ls *LibraryService) GetArtistAlbums(ctx context.Context, artistID int) ([]models.Album, error) {
	query := `
		SELECT a.id, a.name, a.year, a.cover_path,
			   COUNT(DISTINCT s.id) as song_count,
			   COALESCE(SUM(s.duration), 0) as duration
		FROM albums a
		LEFT JOIN songs s ON a.id = s.album_id
		WHERE a.artist_id = $1 OR a.album_artist_id = $1
		GROUP BY a.id, a.name, a.year, a.cover_path
		ORDER BY a.year, a.name
	`

	rows, err := ls.db.QueryContext(ctx, query, artistID)
	if err != nil {
		return nil, fmt.Errorf("failed to query artist albums: %w", err)
	}
	defer rows.Close()

	albums := make([]models.Album, 0)
	for rows.Next() {
		var album models.Album
		err := rows.Scan(&album.ID, &album.Name, &album.Year, &album.CoverPath,
			&album.SongCount, &album.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed to scan album: %w", err)
		}
		albums = append(albums, album)
	}

	return albums, rows.Err()
}

func (ls *LibraryService) GetArtistTopTracks(ctx context.Context, artistID int) ([]models.Song, error) {
	// For now, just return the first 10 songs by the artist ordered by track number
	// In the future, this could be based on play count or other metrics
	query := `
		SELECT s.id, s.title, s.duration,
			   a.id as album_id, a.name as album_name
		FROM songs s
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE s.artist_id = $1
		ORDER BY s.track_number, s.title
		LIMIT 10
	`

	rows, err := ls.db.QueryContext(ctx, query, artistID)
	if err != nil {
		return nil, fmt.Errorf("failed to query artist top tracks: %w", err)
	}
	defer rows.Close()

	songs := make([]models.Song, 0)
	for rows.Next() {
		var song models.Song
		var albumID *int
		var albumName *string
		err := rows.Scan(&song.ID, &song.Title, &song.Duration, &albumID, &albumName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan song: %w", err)
		}

		if albumID != nil && albumName != nil {
			song.Album = &models.Album{
				ID:   *albumID,
				Name: *albumName,
			}
		}

		songs = append(songs, song)
	}

	return songs, rows.Err()
}

func (ls *LibraryService) GetAllSongs(ctx context.Context, limit, offset int, sort string) ([]models.Song, error) {
	query := `
		SELECT s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified, 
			   s.bitrate, s.format, s.cover_path, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name
		FROM songs s
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
	`

	switch sort {
	case "title":
		query += " ORDER BY s.title"
	case "title_desc":
		query += " ORDER BY s.title DESC"
	case "artist":
		query += " ORDER BY ar.name, s.title"
	case "album":
		query += " ORDER BY a.name, s.track_number, s.title"
	case "duration":
		query += " ORDER BY s.duration"
	case "duration_desc":
		query += " ORDER BY s.duration DESC"
	case "date_added":
		query += " ORDER BY s.date_added DESC"
	default:
		query += " ORDER BY s.title"
	}

	query += " LIMIT $1 OFFSET $2"

	return ls.querySongs(ctx, query, limit, offset)
}

func (ls *LibraryService) loadSongsLyrics(ctx context.Context, songs []models.Song) error {
	if len(songs) == 0 {
		return nil
	}

	songIDs := make([]interface{}, len(songs))
	songMap := make(map[int]*models.Song)
	for i, song := range songs {
		songIDs[i] = song.ID
		songMap[song.ID] = &songs[i]
	}

	placeholders := ""
	for i := range songIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(`
		SELECT song_id, id, content, type, source, language, created_at 
		FROM lyrics 
		WHERE song_id IN (%s)
		ORDER BY song_id, type DESC, source`, placeholders)

	rows, err := ls.db.QueryContext(ctx, query, songIDs...)
	if err != nil {
		return fmt.Errorf("failed to query lyrics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lyrics models.Lyrics
		var songID int
		err := rows.Scan(&songID, &lyrics.ID, &lyrics.Content, &lyrics.Type,
			&lyrics.Source, &lyrics.Language, &lyrics.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan lyrics: %w", err)
		}

		if song, exists := songMap[songID]; exists {
			lyrics.SongID = songID
			song.Lyrics = append(song.Lyrics, lyrics)
		}
	}

	return rows.Err()
}
