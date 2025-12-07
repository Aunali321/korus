package services

import (
	"context"
	"fmt"

	"korus/internal/database"
	"korus/internal/models"
)

type UserLibraryService struct {
	db *database.DB
}

func NewUserLibraryService(db *database.DB) *UserLibraryService {
	return &UserLibraryService{db: db}
}

func (uls *UserLibraryService) GetLikedSongs(ctx context.Context, userID int, limit, offset int, sort string) ([]models.Song, error) {
	query := `
		SELECT s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified, 
			   s.bitrate, s.format, s.cover_path, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name,
			   ls.liked_at
		FROM liked_songs ls
		JOIN songs s ON ls.song_id = s.id
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE ls.user_id = $1
	`

	switch sort {
	case "title":
		query += " ORDER BY s.title"
	case "title_desc":
		query += " ORDER BY s.title DESC"
	case "artist":
		query += " ORDER BY ar.name, s.title"
	case "album":
		query += " ORDER BY a.name, s.track_number"
	case "date_added":
		query += " ORDER BY s.date_added DESC"
	case "liked_at":
		query += " ORDER BY ls.liked_at DESC"
	default:
		query += " ORDER BY ls.liked_at DESC"
	}

	query += " LIMIT $2 OFFSET $3"

	rows, err := uls.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query liked songs: %w", err)
	}
	defer rows.Close()

	songs := make([]models.Song, 0)
	for rows.Next() {
		var song models.Song
		var artistName, albumName *string
		var likedAt interface{} // We don't need this in the response but it's in the query

		err := rows.Scan(&song.ID, &song.Title, &song.AlbumID, &song.ArtistID,
			&song.TrackNumber, &song.DiscNumber, &song.Duration,
			&song.FilePath, &song.FileSize, &song.FileModified,
			&song.Bitrate, &song.Format, &song.CoverPath, &song.DateAdded,
			&artistName, &albumName, &likedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan liked song: %w", err)
		}

		if song.ArtistID != nil && artistName != nil {
			song.Artist = &models.Artist{
				ID:   *song.ArtistID,
				Name: *artistName,
			}
		}

		if song.AlbumID != nil && albumName != nil {
			song.Album = &models.Album{
				ID:   *song.AlbumID,
				Name: *albumName,
			}
		}

		songs = append(songs, song)
	}

	return songs, rows.Err()
}

func (uls *UserLibraryService) GetLikedAlbums(ctx context.Context, userID int, limit, offset int) ([]models.Album, error) {
	query := `
		SELECT a.id, a.name, a.artist_id, a.album_artist_id, a.year, 
			   a.musicbrainz_id, a.cover_path, a.date_added,
			   ar.name as artist_name,
			   COUNT(DISTINCT s.id) as song_count,
			   COALESCE(SUM(s.duration), 0) as duration,
			   la.liked_at
		FROM liked_albums la
		JOIN albums a ON la.album_id = a.id
		LEFT JOIN artists ar ON a.artist_id = ar.id
		LEFT JOIN songs s ON a.id = s.album_id
		WHERE la.user_id = $1
		GROUP BY a.id, a.name, a.artist_id, a.album_artist_id, a.year, 
				 a.musicbrainz_id, a.cover_path, a.date_added, ar.name, la.liked_at
		ORDER BY la.liked_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := uls.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query liked albums: %w", err)
	}
	defer rows.Close()

	albums := make([]models.Album, 0)
	for rows.Next() {
		var album models.Album
		var artistName *string
		var likedAt interface{} // We don't need this in the response but it's in the query

		err := rows.Scan(&album.ID, &album.Name, &album.ArtistID, &album.AlbumArtistID,
			&album.Year, &album.MusicBrainzID, &album.CoverPath, &album.DateAdded,
			&artistName, &album.SongCount, &album.Duration, &likedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan liked album: %w", err)
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

func (uls *UserLibraryService) GetFollowedArtists(ctx context.Context, userID int, limit, offset int) ([]models.Artist, error) {
	query := `
		SELECT a.id, a.name, a.sort_name, a.musicbrainz_id,
			   COUNT(DISTINCT al.id) as album_count,
			   COUNT(DISTINCT s.id) as song_count,
			   fa.followed_at
		FROM followed_artists fa
		JOIN artists a ON fa.artist_id = a.id
		LEFT JOIN albums al ON a.id = al.artist_id OR a.id = al.album_artist_id
		LEFT JOIN songs s ON a.id = s.artist_id
		WHERE fa.user_id = $1
		GROUP BY a.id, a.name, a.sort_name, a.musicbrainz_id, fa.followed_at
		ORDER BY fa.followed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := uls.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query followed artists: %w", err)
	}
	defer rows.Close()

	artists := make([]models.Artist, 0)
	for rows.Next() {
		var artist models.Artist
		var followedAt interface{}

		err := rows.Scan(&artist.ID, &artist.Name, &artist.SortName, &artist.MusicBrainzID,
			&artist.AlbumCount, &artist.SongCount, &followedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan followed artist: %w", err)
		}
		artists = append(artists, artist)
	}

	return artists, rows.Err()
}

func (uls *UserLibraryService) LikeSongs(ctx context.Context, userID int, songIDs []int) error {
	if len(songIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO liked_songs (user_id, song_id, liked_at)
		SELECT $1, unnest($2::int[]), NOW()
		ON CONFLICT (user_id, song_id) DO NOTHING
	`

	_, err := uls.db.ExecContext(ctx, query, userID, songIDs)
	if err != nil {
		return fmt.Errorf("failed to like songs: %w", err)
	}

	return nil
}

func (uls *UserLibraryService) UnlikeSongs(ctx context.Context, userID int, songIDs []int) error {
	if len(songIDs) == 0 {
		return nil
	}

	query := `DELETE FROM liked_songs WHERE user_id = $1 AND song_id = ANY($2)`

	_, err := uls.db.ExecContext(ctx, query, userID, songIDs)
	if err != nil {
		return fmt.Errorf("failed to unlike songs: %w", err)
	}

	return nil
}

func (uls *UserLibraryService) LikeAlbum(ctx context.Context, userID, albumID int) error {
	query := `
		INSERT INTO liked_albums (user_id, album_id, liked_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, album_id) DO NOTHING
	`

	_, err := uls.db.ExecContext(ctx, query, userID, albumID)
	if err != nil {
		return fmt.Errorf("failed to like album: %w", err)
	}

	return nil
}

func (uls *UserLibraryService) UnlikeAlbum(ctx context.Context, userID, albumID int) error {
	query := `DELETE FROM liked_albums WHERE user_id = $1 AND album_id = $2`

	result, err := uls.db.ExecContext(ctx, query, userID, albumID)
	if err != nil {
		return fmt.Errorf("failed to unlike album: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("album not found in user's liked albums")
	}

	return nil
}

func (uls *UserLibraryService) FollowArtist(ctx context.Context, userID, artistID int) error {
	query := `
		INSERT INTO followed_artists (user_id, artist_id, followed_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, artist_id) DO NOTHING
	`

	_, err := uls.db.ExecContext(ctx, query, userID, artistID)
	if err != nil {
		return fmt.Errorf("failed to follow artist: %w", err)
	}

	return nil
}

func (uls *UserLibraryService) UnfollowArtist(ctx context.Context, userID, artistID int) error {
	query := `DELETE FROM followed_artists WHERE user_id = $1 AND artist_id = $2`

	result, err := uls.db.ExecContext(ctx, query, userID, artistID)
	if err != nil {
		return fmt.Errorf("failed to unfollow artist: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("artist not found in user's followed artists")
	}

	return nil
}

func (uls *UserLibraryService) CheckSongLiked(ctx context.Context, userID, songID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM liked_songs WHERE user_id = $1 AND song_id = $2)`

	var liked bool
	err := uls.db.QueryRowContext(ctx, query, userID, songID).Scan(&liked)
	if err != nil {
		return false, fmt.Errorf("failed to check song liked status: %w", err)
	}

	return liked, nil
}

func (uls *UserLibraryService) CheckAlbumLiked(ctx context.Context, userID, albumID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM liked_albums WHERE user_id = $1 AND album_id = $2)`

	var liked bool
	err := uls.db.QueryRowContext(ctx, query, userID, albumID).Scan(&liked)
	if err != nil {
		return false, fmt.Errorf("failed to check album liked status: %w", err)
	}

	return liked, nil
}

func (uls *UserLibraryService) CheckArtistFollowed(ctx context.Context, userID, artistID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM followed_artists WHERE user_id = $1 AND artist_id = $2)`

	var followed bool
	err := uls.db.QueryRowContext(ctx, query, userID, artistID).Scan(&followed)
	if err != nil {
		return false, fmt.Errorf("failed to check artist followed status: %w", err)
	}

	return followed, nil
}
