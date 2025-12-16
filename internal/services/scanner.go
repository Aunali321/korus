package services

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dhowden/tag"
	"github.com/fsnotify/fsnotify"
)

// lrcPattern matches LRC timestamp format [mm:ss] or [mm:ss.xx]
var lrcPattern = regexp.MustCompile(`\[\d{2}:\d{2}`)

func isLRCFormat(text string) bool {
	return lrcPattern.MatchString(text)
}

type ScanStatus struct {
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
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
	workers        int
	coverCachePath string
	autoPlaylists  bool
}

func NewScannerService(db *sql.DB, mediaRoot, ffprobePath, ffmpegPath, exclude string, scanEmbeddedCover bool, watch bool, workers int, coverCachePath string, autoPlaylists bool) *ScannerService {
	var re *regexp.Regexp
	if exclude != "" {
		re = regexp.MustCompile(exclude)
	}
	if workers < 1 {
		workers = 8
	}
	// Default cover cache path if not specified
	if coverCachePath == "" {
		coverCachePath = "./cache/covers"
	}
	return &ScannerService{
		db:             db,
		mediaRoot:      mediaRoot,
		excludePattern: re,
		scanEmbedded:   scanEmbeddedCover,
		ffprobePath:    ffprobePath,
		ffmpegPath:     ffmpegPath,
		watchEnabled:   watch,
		workers:        workers,
		coverCachePath: coverCachePath,
		autoPlaylists:  autoPlaylists,
	}
}

func (s *ScannerService) IsScanning() bool { return atomic.LoadInt32(&s.scanning) == 1 }

func (s *ScannerService) StartScan(ctx context.Context) (int64, error) {
	if !atomic.CompareAndSwapInt32(&s.scanning, 0, 1) {
		return 0, errors.New("scan already running")
	}

	// Insert scan status row immediately so it's visible to status queries
	res, err := s.db.ExecContext(ctx, `INSERT INTO scan_status(status, progress, total) VALUES ('running', 0, 0)`)
	if err != nil {
		atomic.StoreInt32(&s.scanning, 0)
		return 0, fmt.Errorf("insert scan status: %w", err)
	}
	scanID, _ := res.LastInsertId()

	// Run the actual scan in a goroutine
	go s.runScan(scanID)

	return scanID, nil
}

func (s *ScannerService) runScan(scanID int64) {
	defer atomic.StoreInt32(&s.scanning, 0)
	ctx := context.Background()

	var files []string
	var playlists []string
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
		} else if s.autoPlaylists && isPlaylistFile(path) {
			playlists = append(playlists, path)
		}
		return nil
	})
	if err != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE scan_status SET status='failed' WHERE id=?`, scanID)
		return
	}

	// Update total count now that we know it
	_, _ = s.db.ExecContext(ctx, `UPDATE scan_status SET total=? WHERE id=?`, len(files), scanID)

	var mu sync.Mutex
	seenSongs := map[int64]struct{}{}
	seenAlbums := map[int64]struct{}{}
	seenArtists := map[int64]struct{}{}

	var processedCount int64
	var currentFile atomic.Value
	currentFile.Store("")

	progressDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		lastReported := int64(-1)
		for {
			select {
			case <-progressDone:
				count := atomic.LoadInt64(&processedCount)
				file, _ := currentFile.Load().(string)
				_, _ = s.db.ExecContext(ctx, `UPDATE scan_status SET current_file = ?, progress = ? WHERE id = ?`,
					file, count, scanID)
				return
			case <-ticker.C:
				count := atomic.LoadInt64(&processedCount)
				if count != lastReported {
					file, _ := currentFile.Load().(string)
					_, _ = s.db.ExecContext(ctx, `UPDATE scan_status SET current_file = ?, progress = ? WHERE id = ?`,
						file, count, scanID)
					lastReported = count
				}
			}
		}
	}()

	fileChan := make(chan string, len(files))
	var wg sync.WaitGroup

	for w := 0; w < s.workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				localSongs := map[int64]struct{}{}
				localAlbums := map[int64]struct{}{}
				localArtists := map[int64]struct{}{}

				currentFile.Store(file)

				if err := s.ingestFile(ctx, file, localSongs, localAlbums, localArtists); err != nil {
					atomic.AddInt64(&processedCount, 1)
				} else {
					mu.Lock()
					for id := range localSongs {
						seenSongs[id] = struct{}{}
					}
					for id := range localAlbums {
						seenAlbums[id] = struct{}{}
					}
					for id := range localArtists {
						seenArtists[id] = struct{}{}
					}
					mu.Unlock()

					atomic.AddInt64(&processedCount, 1)
				}
			}
		}()
	}

	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	wg.Wait()

	close(progressDone)
	time.Sleep(100 * time.Millisecond)

	if err := s.cleanup(ctx, seenSongs, seenAlbums, seenArtists); err != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE scan_status SET status='failed' WHERE id=?`, scanID)
		return
	}

	if s.autoPlaylists {
		s.importPlaylists(ctx, playlists)
	}

	now := time.Now()
	_, _ = s.db.ExecContext(ctx, `UPDATE scan_status SET status='completed', progress=?, completed_at=? WHERE id=?`, len(files), now, scanID)
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
	audioMeta := s.probe(path)

	var existingLyrics, existingSynced, existingMBID string
	_ = s.db.QueryRowContext(ctx, `SELECT lyrics, lyrics_synced, COALESCE(mbid, '') FROM songs WHERE file_path = ?`, path).Scan(&existingLyrics, &existingSynced, &existingMBID)

	rawLyrics := meta.Lyrics()
	lyrics := existingLyrics
	lyricsSynced := existingSynced

	// Detect LRC format and store in correct column
	if rawLyrics != "" {
		if isLRCFormat(rawLyrics) {
			lyricsSynced = rawLyrics
		} else {
			lyrics = rawLyrics
		}
	}
	mbid := existingMBID

	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO songs(id, album_id, title, track_number, duration_ms, sample_rate, bit_depth, channels, file_path, lyrics, lyrics_synced, mbid)
		VALUES (
			COALESCE((SELECT id FROM songs WHERE file_path = ?), NULL),
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, '')
		)
	`, path, albumID, title, trackNo, audioMeta.DurationMs, audioMeta.SampleRate, audioMeta.BitDepth, audioMeta.Channels, path, lyrics, lyricsSynced, mbid)
	if err != nil {
		return fmt.Errorf("insert song: %w", err)
	}

	var songID int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM songs WHERE file_path = ?`, path).Scan(&songID); err == nil {
		seenSongs[songID] = struct{}{}
		// Delete old FTS entry and insert new one (FTS5 contentless tables don't support ON CONFLICT)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM songs_fts WHERE rowid = ?`, songID)
		_, _ = s.db.ExecContext(ctx, `INSERT INTO songs_fts (rowid, song_id, title, artist_name, album_title)
			VALUES (?, ?, ?, ?, ?)`, songID, songID, title, artistName, albumTitle)
	}

	if s.scanEmbedded {
		coverPath, err := s.extractCover(ctx, path, albumID)
		if err == nil && coverPath != "" {
			_, _ = s.db.ExecContext(ctx, `UPDATE albums SET cover_path = ? WHERE id = ?`, coverPath, albumID)
		}
	}

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
	var currentFile sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT status, progress, started_at, completed_at, total, current_file FROM scan_status ORDER BY id DESC LIMIT 1
	`).Scan(&st.Status, &st.Progress, &st.StartedAt, &st.CompletedAt, &st.Total, &currentFile)
	if err != nil {
		return st, err
	}
	st.CurrentFile = currentFile.String
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
			// Skip hidden directories
			if strings.HasPrefix(d.Name(), ".") && path != s.mediaRoot {
				return fs.SkipDir
			}
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
			// Ignore events from hidden files/directories and database files
			if strings.Contains(ev.Name, "/.") || strings.HasSuffix(ev.Name, ".db") {
				continue
			}
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
				debounce.Reset(time.Second * 5)
			}
		case <-debounce.C:
			// Only trigger if not already scanning
			if !s.IsScanning() {
				go s.StartScan(context.Background())
			}
		case err := <-w.Errors:
			return err
		}
	}
}

type audioMetadata struct {
	DurationMs int
	SampleRate int
	BitDepth   int
	Channels   int
}

func (s *ScannerService) probe(path string) audioMetadata {
	cmd := exec.Command(s.ffprobePath, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()
	var data struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType        string `json:"codec_type"`
			SampleRate       string `json:"sample_rate"`
			Channels         int    `json:"channels"`
			BitsPerSample    int    `json:"bits_per_sample"`
			BitsPerRawSample string `json:"bits_per_raw_sample"`
		} `json:"streams"`
	}
	meta := audioMetadata{}
	if err := json.Unmarshal(out.Bytes(), &data); err == nil {
		if dur, err := strconv.ParseFloat(data.Format.Duration, 64); err == nil {
			meta.DurationMs = int(dur * 1000)
		}
		for _, stream := range data.Streams {
			if stream.CodecType == "audio" {
				meta.SampleRate = parseStringToInt(stream.SampleRate)
				meta.Channels = stream.Channels
				// bit_depth can come from bits_per_sample or bits_per_raw_sample
				// For lossy formats (MP3, AAC, etc.) these are often 0 - default to 16-bit
				if stream.BitsPerSample > 0 {
					meta.BitDepth = stream.BitsPerSample
				} else if stream.BitsPerRawSample != "" {
					meta.BitDepth = parseStringToInt(stream.BitsPerRawSample)
				}
				if meta.BitDepth == 0 {
					meta.BitDepth = 16 // Default to 16-bit for lossy formats
				}
				break
			}
		}
	}
	return meta
}

func parseStringToInt(v string) int {
	i, _ := strconv.Atoi(v)
	return i
}

func (s *ScannerService) extractCover(ctx context.Context, src string, albumID int64) (string, error) {
	if err := os.MkdirAll(s.coverCachePath, 0o755); err != nil {
		return "", err
	}
	outPath := filepath.Join(s.coverCachePath, fmt.Sprintf("%d.jpg", albumID))
	// Extract and resize to 500x500, compress as JPEG with quality 85
	cmd := exec.CommandContext(ctx, s.ffmpegPath,
		"-y", "-i", src,
		"-an",                                                                                       // No audio
		"-vf", "scale=500:500:force_original_aspect_ratio=decrease,pad=500:500:(ow-iw)/2:(oh-ih)/2", // Resize and pad to 500x500
		"-frames:v", "1", // Just one frame
		"-q:v", "2", // JPEG quality (2 is high quality, smaller file)
		outPath)
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

func isPlaylistFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".m3u" || ext == ".m3u8"
}

func (s *ScannerService) importPlaylists(ctx context.Context, m3uFiles []string) {
	seenPaths := make(map[string]struct{})
	for _, path := range m3uFiles {
		seenPaths[path] = struct{}{}
		if err := s.importM3U(ctx, path); err != nil {
			log.Printf("import playlist %s: %v", path, err)
		}
	}
	s.cleanupPlaylists(ctx, seenPaths)
}

func (s *ScannerService) importM3U(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open m3u: %w", err)
	}
	defer f.Close()

	var trackPaths []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		trackPath := line
		if !filepath.IsAbs(trackPath) {
			trackPath = filepath.Join(filepath.Dir(path), trackPath)
		}
		trackPath = filepath.Clean(trackPath)
		trackPaths = append(trackPaths, trackPath)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan m3u: %w", err)
	}

	if len(trackPaths) == 0 {
		return nil
	}

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	var adminID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE role = 'admin' ORDER BY id LIMIT 1`).Scan(&adminID)
	if err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}

	var playlistID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM playlists WHERE source_path = ?`, path).Scan(&playlistID)
	if errors.Is(err, sql.ErrNoRows) {
		res, err := s.db.ExecContext(ctx, `INSERT INTO playlists(user_id, name, description, public, source_path) VALUES(?, ?, '', 1, ?)`,
			adminID, name, path)
		if err != nil {
			return fmt.Errorf("create playlist: %w", err)
		}
		playlistID, _ = res.LastInsertId()
	} else if err != nil {
		return fmt.Errorf("query playlist: %w", err)
	} else {
		_, _ = s.db.ExecContext(ctx, `UPDATE playlists SET name = ? WHERE id = ?`, name, playlistID)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM playlist_songs WHERE playlist_id = ?`, playlistID)
	}

	for i, trackPath := range trackPaths {
		var songID int64
		err := s.db.QueryRowContext(ctx, `SELECT id FROM songs WHERE file_path = ?`, trackPath).Scan(&songID)
		if err != nil {
			continue
		}
		_, _ = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO playlist_songs(playlist_id, song_id, position) VALUES(?, ?, ?)`,
			playlistID, songID, i+1)
	}

	return nil
}

func (s *ScannerService) cleanupPlaylists(ctx context.Context, seenPaths map[string]struct{}) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, source_path FROM playlists WHERE source_path IS NOT NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var sourcePath string
		if err := rows.Scan(&id, &sourcePath); err != nil {
			continue
		}
		if _, ok := seenPaths[sourcePath]; !ok {
			_, _ = s.db.ExecContext(ctx, `DELETE FROM playlists WHERE id = ?`, id)
		}
	}
}
