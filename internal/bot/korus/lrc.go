package korus

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Line is one lyric line and the moment it is sung.
type Line struct {
	At   time.Duration
	Text string
}

// A stamp is [mm:ss], [mm:ss.xx] or [mm:ss.xxx]. Metadata tags like [ar:Name]
// have no digits before the colon and are left alone.
var lrcStamp = regexp.MustCompile(`\[(\d{1,3}):(\d{2})(?:[.:](\d{1,3}))?\]`)

// Timed parses LRC lyrics into lines ordered by timestamp. A source line may
// carry several stamps, which repeats its text at each one. Lines with no text
// are kept: LRC uses them to clear the display through instrumental passages.
func (l Lyrics) Timed() []Line {
	var lines []Line
	for _, raw := range strings.Split(l.Synced, "\n") {
		stamps := lrcStamp.FindAllStringSubmatch(raw, -1)
		if len(stamps) == 0 {
			continue
		}
		text := strings.TrimSpace(raw[strings.LastIndex(raw, "]")+1:])
		for _, stamp := range stamps {
			lines = append(lines, Line{At: stampAt(stamp), Text: text})
		}
	}
	slices.SortStableFunc(lines, func(a, b Line) int { return int(a.At - b.At) })
	return lines
}

func stampAt(stamp []string) time.Duration {
	minutes, _ := strconv.Atoi(stamp[1])
	seconds, _ := strconv.Atoi(stamp[2])
	at := time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second

	fraction := stamp[3]
	if fraction == "" {
		return at
	}
	// The fraction is a decimal part of a second, so each digit is a tenth of
	// the last: .5 is 500ms, .06 is 60ms, .060 is also 60ms.
	value, _ := strconv.Atoi(fraction)
	scale := time.Second
	for range fraction {
		scale /= 10
	}
	return at + time.Duration(value)*scale
}

// LineAt returns the index of the line being sung at elapsed, or -1 before the
// first one.
func LineAt(lines []Line, elapsed time.Duration) int {
	index := -1
	for i, line := range lines {
		if line.At > elapsed {
			break
		}
		index = i
	}
	return index
}
