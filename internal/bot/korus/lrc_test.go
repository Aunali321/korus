package korus

import (
	"testing"
	"time"
)

func TestTimedParsesRealLRC(t *testing.T) {
	lyrics := Lyrics{Synced: "[ar:TV Girl]\n[00:02.06]Know I miss my brother\n[00:05.57]I know\n[01:07.5]Late line\n[02:00]No fraction\n"}
	lines := lyrics.Timed()

	want := []Line{
		{2*time.Second + 60*time.Millisecond, "Know I miss my brother"},
		{5*time.Second + 570*time.Millisecond, "I know"},
		{67*time.Second + 500*time.Millisecond, "Late line"},
		{2 * time.Minute, "No fraction"},
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d: %+v", len(lines), len(want), lines)
	}
	for i, line := range lines {
		if line != want[i] {
			t.Errorf("line %d = %+v, want %+v", i, line, want[i])
		}
	}
}

// One source line may carry several stamps, and a stamp with no text clears the
// display through an instrumental break.
func TestTimedRepeatsAndBlanks(t *testing.T) {
	lines := Lyrics{Synced: "[00:10.00][00:30.00]Chorus\n[00:20.00]\n"}.Timed()
	want := []Line{
		{10 * time.Second, "Chorus"},
		{20 * time.Second, ""},
		{30 * time.Second, "Chorus"},
	}
	if len(lines) != 3 {
		t.Fatalf("got %+v", lines)
	}
	for i, line := range lines {
		if line != want[i] {
			t.Errorf("line %d = %+v, want %+v", i, line, want[i])
		}
	}
}

func TestTimedIgnoresUnsynced(t *testing.T) {
	if lines := (Lyrics{Lyrics: "plain words\nmore words"}).Timed(); len(lines) != 0 {
		t.Fatalf("plain lyrics produced %+v", lines)
	}
}

func TestLineAt(t *testing.T) {
	lines := Lyrics{Synced: "[00:05.00]one\n[00:10.00]two\n[00:20.00]three\n"}.Timed()
	for _, tc := range []struct {
		elapsed time.Duration
		want    int
	}{
		{0, -1},
		{4 * time.Second, -1},
		{5 * time.Second, 0},
		{9 * time.Second, 0},
		{10 * time.Second, 1},
		{19 * time.Second, 1},
		{60 * time.Second, 2},
	} {
		if got := LineAt(lines, tc.elapsed); got != tc.want {
			t.Errorf("LineAt(%s) = %d, want %d", tc.elapsed, got, tc.want)
		}
	}
}
