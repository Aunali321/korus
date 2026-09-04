package player

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"testing"
	"time"
)

// TestOggReaderFrames decodes real FFmpeg output: five seconds of tone must
// arrive as 20ms Opus packets with the container headers stripped.
func TestOggReaderFrames(t *testing.T) {
	cmd := exec.Command("ffmpeg",
		"-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=5",
		"-c:a", "libopus", "-b:a", "128k", "-ar", "48000", "-ac", "2",
		"-frame_duration", "20", "-f", "ogg", "pipe:1",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Skipf("ffmpeg unavailable: %v (%s)", err, stderr.String())
	}

	reader := newOggReader(&stdout)
	packets := 0
	for {
		packet, err := reader.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("next packet: %v", err)
		}
		if len(packet) == 0 {
			t.Fatal("empty opus packet")
		}
		if isOpusHeader(packet) {
			t.Fatal("container header leaked into the opus stream")
		}
		packets++
	}

	got := time.Duration(packets) * frameDuration
	if got < 4500*time.Millisecond || got > 5500*time.Millisecond {
		t.Fatalf("decoded %s of audio from %d packets, want about 5s", got, packets)
	}
}
