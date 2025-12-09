package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dhowden/tag"
	"github.com/fsnotify/fsnotify"
)

type ScanStatus struct {
	Status      string     `json:"status"`
	Progress    float64    `json:"progress"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CurrentFile string     `json:"current_file,omitempty"`
	Total       int        `json:"total"`
}

type ScannerService struct {
	db             *sql.DB
	mediaRoot      string
	excludePattern *regexp.Regexp
	scanEmbedded   bool
	ffprobePath    string
	ffmpegPath     string
	scanning       int32
	watchEnabled   bool
}

func NewScannerService(db *sql.DB, mediaRoot, ffprobePath, ffmpegPath, exclude string, scanEmbeddedCover bool, watch bool) *ScannerService {
	var re *regexp.Regexp
	if exclude != "" {
		re = regexp.MustCompile(exclude)
	}
	return &ScannerService{
		db:             db,
		mediaRoot:      mediaRoot,
		excludePattern: re,
		scanEmbedded:   scanEmbeddedCover,
		ffprobePath:    ffprobePath,
		ffmpegPath:     ffmpegPath,
		watchEnabled:   watch,
	}
}

func (s *ScannerService) IsScanning() bool { return atomic.LoadInt32(&s.scanning) == 1 }

func (s *ScannerService) StartScan(ctx context.Context) (int64, error) {
	if !atomic.CompareAndSwapInt32(&s.scanning, 0, 1) {
		return 0, errors.New("scan already running")
	}
	defer atomic.StoreInt32(&s.scanning, 0)

	var files []string
	err := filepath.WalkDir(s.mediaRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if s.excludePattern != nil && s.excludePattern.MatchString(path) {
				return fs.SkipDir
			}
			return nil
		}
		if s.excludePattern != nil && s.excludePattern.MatchString(path) {
			return nil
		}
		if isAudioFile(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walk media root: %w", err)
	}

	res, err := s.db.ExecContext(ctx, `INSERT INTO scan_status(status, progress, total) VALUES ('running', 0, ?)`, len(files))
	if err != nil {
		return 0, fmt.Errorf("insert scan status: %w", err)
	}
	scanID, _ := res.LastInsertId()

	seenSongs := map[int64]struct{}{}
	seenAlbums := map[int64]struct{}{}
	seenArtists := map[int64]struct{}{}

	for i, file := range files {
		if err := s.ingestFile(ctx, file, seenSongs, seenAlbums, seenArtists); err != nil {
			// continue but record error in status current_file
			_, _ = s.db.ExecContext(ctx, `UPDATE scan_status SET current_file = ?, progress = ? WHERE id = ?`, fmt.Sprintf("error:%s", err.Error()), progress(i, len(files)), scanID)
			continue
		}
		_, _ = s.db.ExecContext(ctx, `UPDATE scan_status SET current_file = ?, progress = ? WHERE id = ?`, file, progress(i+1, len(files)), scanID)
	}

	if err := s.cleanup(ctx, seenSongs, seenAlbums, seenArtists); err != nil {
		return 0, err
	}

	now := time.Now()
	_, _ = s.db.ExecContext(ctx, `UPDATE scan_status SET status='completed', progress=1, completed_at=? WHERE id=?`, now, scanID)
	return scanID, nil
}

func (s *ScannerService) ingestFile(ctx context.Context, path string, seenSongs map[int64]struct{}, seenAlbums map[int64]struct{}, seenArtists map[int64]struct{}) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	meta, err := tag.ReadFrom(f)
	if err != nil {
		return fmt.Errorf("read tags: %w", err)
	}
	artistName := meta.Artist()
	albumTitle := meta.Album()
	title := meta.Title()
	if artistName == "" || albumTitle == "" || title == "" {
		return errors.New("missing metadata")
	}
	var artistID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM artists WHERE name = ?`, artistName).Scan(&artistID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			res, err := s.db.ExecContext(ctx, `INSERT INTO artists(name) VALUES (?)`, artistName)
			if err != nil {
				return err
			}
			artistID, _ = res.LastInsertId()
		} else {
			return err
		}
	}
	seenArtists[artistID] = struct{}{}

	var albumID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM albums WHERE artist_id = ? AND title = ?`, artistID, albumTitle).Scan(&albumID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			year := meta.Year()
			res, err := s.db.ExecContext(ctx, `INSERT INTO albums(artist_id, title, year) VALUES (?, ?, ?)`, artistID, albumTitle, year)
			if err != nil {
				return err
			}
			albumID, _ = res.LastInsertId()
		} else {
			return err
		}
	}
	seenAlbums[albumID] = struct{}{}

	trackNo, _ := meta.Track()
	dur, bitrate := s.probe(path)

	var existingLyrics, existingSynced, existingMBID string
	_ = s.db.QueryRowContext(ctx, `SELECT lyrics, lyrics_synced, COALESCE(mbid, '') FROM songs WHERE file_path = ?`, path).Scan(&existingLyrics, &existingSynced, &existingMBID)

	lyrics := meta.Lyrics()
	if lyrics == "" {
		lyrics = existingLyrics
	}
	lyricsSynced := existingSynced
	mbid := existingMBID

	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO songs(id, album_id, title, track_number, duration, file_path, lyrics, lyrics_synced, mbid)
		VALUES (
			COALESCE((SELECT id FROM songs WHERE file_path = ?), NULL),
			?, ?, ?, ?, ?, ?, ?, NULLIF(?, '')
		)
	`, path, albumID, title, trackNo, dur, path, lyrics, lyricsSynced, mbid)
	if err != nil {
		return fmt.Errorf("insert song: %w", err)
	}

	var songID int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM songs WHERE file_path = ?`, path).Scan(&songID); err == nil {
		seenSongs[songID] = struct{}{}
		_, _ = s.db.ExecContext(ctx, `INSERT INTO songs_fts (rowid, song_id, title, artist_name, album_title)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(rowid) DO UPDATE SET title=excluded.title, artist_name=excluded.artist_name, album_title=excluded.album_title
		`, songID, songID, title, artistName, albumTitle)
	}

	if s.scanEmbedded {
		coverPath, err := s.extractCover(ctx, path, albumID)
		if err == nil && coverPath != "" {
			_, _ = s.db.ExecContext(ctx, `UPDATE albums SET cover_path = ? WHERE id = ?`, coverPath, albumID)
		}
	}

	_ = bitrate // currently unused; placeholder for future schema
	return nil
}

func (s *ScannerService) cleanup(ctx context.Context, songs map[int64]struct{}, albums map[int64]struct{}, artists map[int64]struct{}) error {
	// remove songs not seen
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM songs`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			if _, ok := songs[id]; !ok {
				_, _ = s.db.ExecContext(ctx, `DELETE FROM songs WHERE id = ?`, id)
				_, _ = s.db.ExecContext(ctx, `DELETE FROM songs_fts WHERE rowid = ?`, id)
			}
		}
	}

	// remove albums with no songs
	albRows, err := s.db.QueryContext(ctx, `SELECT id FROM albums`)
	if err != nil {
		return err
	}
	defer albRows.Close()
	for albRows.Next() {
		var id int64
		if err := albRows.Scan(&id); err == nil {
			if _, ok := albums[id]; !ok {
				_, _ = s.db.ExecContext(ctx, `DELETE FROM albums WHERE id = ?`, id)
			}
		}
	}

	// remove artists without albums
	artRows, err := s.db.QueryContext(ctx, `SELECT id FROM artists`)
	if err != nil {
		return err
	}
	defer artRows.Close()
	for artRows.Next() {
		var id int64
		if err := artRows.Scan(&id); err == nil {
			if _, ok := artists[id]; !ok {
				_, _ = s.db.ExecContext(ctx, `DELETE FROM artists WHERE id = ?`, id)
			}
		}
	}
	return nil
}

func (s *ScannerService) Status(ctx context.Context) (ScanStatus, error) {
	var st ScanStatus
	err := s.db.QueryRowContext(ctx, `
		SELECT status, progress, started_at, completed_at, total, current_file FROM scan_status ORDER BY id DESC LIMIT 1
	`).Scan(&st.Status, &st.Progress, &st.StartedAt, &st.CompletedAt, &st.Total, &st.CurrentFile)
	if err != nil {
		return st, err
	}
	return st, nil
}

func (s *ScannerService) Watch(ctx context.Context) error {
	if !s.watchEnabled {
		return nil
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	filepath.WalkDir(s.mediaRoot, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = w.Add(path)
		}
		return nil
	})
	debounce := time.NewTimer(time.Second * 5)
	debounce.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-w.Events:
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
				debounce.Reset(time.Second * 5)
			}
		case <-debounce.C:
			go s.StartScan(context.Background())
		case err := <-w.Errors:
			return err
		}
	}
}

func (s *ScannerService) probe(path string) (int, int) {
	cmd := exec.Command(s.ffprobePath, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()
	var data struct {
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out.Bytes(), &data); err == nil {
		if dur, err := parseFloatToInt(data.Format.Duration); err == nil {
			return dur, parseStringToInt(data.Format.BitRate)
		}
	}
	return 0, 0
}

func parseFloatToInt(v string) (int, error) {
	if v == "" {
		return 0, errors.New("empty")
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}

func parseStringToInt(v string) int {
	i, _ := strconv.Atoi(v)
	return i
}

func (s *ScannerService) extractCover(ctx context.Context, src string, albumID int64) (string, error) {
	outDir := filepath.Join(s.mediaRoot, ".covers")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	outPath := filepath.Join(outDir, fmt.Sprintf("%d.jpg", albumID))
	cmd := exec.CommandContext(ctx, s.ffmpegPath, "-y", "-i", src, "-an", "-vcodec", "copy", "-map", "0:v:0", "-frames:v", "1", outPath)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	if _, err := os.Stat(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func isAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3", ".flac", ".m4a", ".aac", ".ogg", ".wav", ".opus":
		return true
	default:
		return false
	}
}

func progress(done, total int) float64 {
	if total == 0 {
		return 1
	}
	return float64(done) / float64(total)
}
