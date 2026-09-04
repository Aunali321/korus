package player

import (
	"bytes"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

const frameDuration = 20 * time.Millisecond

// stream transcodes one track to Opus and hands frames to the voice connection.
// Discord pulls a frame every 20ms, so pause is a per-frame gate rather than a
// process signal: returning no frame makes the sender emit silence and stop
// speaking, and the elapsed counter freezes with it.
type stream struct {
	cmd    *exec.Cmd
	ogg    *oggReader
	stderr *bytes.Buffer

	frames atomic.Int64
	paused atomic.Bool
	done   chan struct{}
	finish func()
}

// newStream starts FFmpeg on the authenticated download endpoint. The token
// travels in an HTTP header so it never lands in a process listing URL.
func newStream(ffmpegPath, url, bearer string) (*stream, error) {
	cmd := exec.Command(ffmpegPath,
		"-loglevel", "error",
		"-headers", "Authorization: Bearer "+bearer+"\r\n",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-i", url,
		"-vn",
		"-c:a", "libopus",
		"-b:a", "128k",
		"-ar", "48000",
		"-ac", "2",
		"-frame_duration", "20",
		"-application", "audio",
		"-f", "ogg",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg stdout: %w", err)
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}
	s := &stream{
		cmd:    cmd,
		ogg:    newOggReader(stdout),
		stderr: stderr,
		done:   make(chan struct{}),
	}
	// Kill and Wait both report on a process that already exited on its own,
	// which is the normal end of a track.
	s.finish = sync.OnceFunc(func() {
		close(s.done)
		if s.cmd.Process != nil {
			s.cmd.Process.Kill()
		}
		s.cmd.Wait()
	})
	return s, nil
}

// ProvideOpusFrame implements voice.OpusFrameProvider.
func (s *stream) ProvideOpusFrame() ([]byte, error) {
	if s.paused.Load() {
		return nil, nil
	}
	packet, err := s.ogg.next()
	if err != nil {
		s.finish()
		return nil, nil
	}
	s.frames.Add(1)
	return packet, nil
}

// Close implements voice.OpusFrameProvider and is also how skip and stop end a track.
func (s *stream) Close() { s.finish() }

func (s *stream) setPaused(paused bool) { s.paused.Store(paused) }

func (s *stream) isPaused() bool { return s.paused.Load() }

// elapsed counts delivered frames, so it tracks audible playback rather than wall clock.
func (s *stream) elapsed() time.Duration {
	return time.Duration(s.frames.Load()) * frameDuration
}

func (s *stream) err() string { return s.stderr.String() }
