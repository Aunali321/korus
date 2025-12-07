package services

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pemistahl/lingua-go"
)

type LRCLine struct {
	Time    int    `json:"time"`    // Time in milliseconds
	TimeStr string `json:"timeStr"` // Original time string [mm:ss.xx]
	Text    string `json:"text"`    // Lyrics text
}

type LRCMetadata struct {
	Title    string `json:"title"`    // [ti:title]
	Artist   string `json:"artist"`   // [ar:artist]
	Album    string `json:"album"`    // [al:album]
	By       string `json:"by"`       // [by:creator]
	Offset   int    `json:"offset"`   // [offset:+/-offset] in milliseconds
	Length   string `json:"length"`   // [length:mm:ss]
	Language string `json:"language"` // [la:language]
}

type LRCDocument struct {
	Metadata LRCMetadata `json:"metadata"`
	Lines    []LRCLine   `json:"lines"`
}

type LRCParser struct {
	metadataRegex  *regexp.Regexp
	timestampRegex *regexp.Regexp
}

func NewLRCParser() *LRCParser {
	return &LRCParser{
		metadataRegex:  regexp.MustCompile(`^\[([a-zA-Z]+):(.+?)\]$`),
		timestampRegex: regexp.MustCompile(`^\[(\d{1,2}):(\d{2})\.(\d{2})\](.*)$`),
	}
}

func (p *LRCParser) Parse(reader io.Reader) (*LRCDocument, error) {
	doc := &LRCDocument{
		Lines: make([]LRCLine, 0),
	}

	scanner := bufio.NewScanner(reader)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if p.parseMetadata(line, &doc.Metadata) {
			continue
		}

		if lrcLine, ok := p.parseTimestamp(line); ok {
			doc.Lines = append(doc.Lines, lrcLine)
			continue
		}

		// If it doesn't match either pattern, it might be a plain text line
		// We can choose to include it as a line without timestamp or skip it
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}

		// Plain text line without timestamp - include it with time 0
		doc.Lines = append(doc.Lines, LRCLine{
			Time:    0,
			TimeStr: "[00:00.00]",
			Text:    line,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading LRC file: %w", err)
	}

	sort.Slice(doc.Lines, func(i, j int) bool {
		return doc.Lines[i].Time < doc.Lines[j].Time
	})

	return doc, nil
}

// parseMetadata parses metadata lines like [ti:title]
func (p *LRCParser) parseMetadata(line string, metadata *LRCMetadata) bool {
	matches := p.metadataRegex.FindStringSubmatch(line)
	if len(matches) != 3 {
		return false
	}

	tag := strings.ToLower(matches[1])
	value := strings.TrimSpace(matches[2])

	switch tag {
	case "ti", "title":
		metadata.Title = value
	case "ar", "artist":
		metadata.Artist = value
	case "al", "album":
		metadata.Album = value
	case "by":
		metadata.By = value
	case "offset":
		if offset, err := strconv.Atoi(value); err == nil {
			metadata.Offset = offset
		}
	case "length":
		metadata.Length = value
	case "la", "lang", "language":
		metadata.Language = value
	default:
		return true
	}

	return true
}

func (p *LRCParser) parseTimestamp(line string) (LRCLine, bool) {
	matches := p.timestampRegex.FindStringSubmatch(line)
	if len(matches) != 5 {
		return LRCLine{}, false
	}

	minutes, err1 := strconv.Atoi(matches[1])
	seconds, err2 := strconv.Atoi(matches[2])
	centiseconds, err3 := strconv.Atoi(matches[3])

	if err1 != nil || err2 != nil || err3 != nil {
		return LRCLine{}, false
	}

	totalMilliseconds := (minutes*60+seconds)*1000 + centiseconds*10

	timeStr := fmt.Sprintf("[%02d:%02d.%02d]", minutes, seconds, centiseconds)

	text := strings.TrimSpace(matches[4])

	return LRCLine{
		Time:    totalMilliseconds,
		TimeStr: timeStr,
		Text:    text,
	}, true
}

func (doc *LRCDocument) ToJSON() (string, error) {
	var result strings.Builder

	result.WriteString("{")
	result.WriteString(fmt.Sprintf(`"metadata":{"title":"%s","artist":"%s","album":"%s","by":"%s","offset":%d,"length":"%s","language":"%s"},`,
		escapeJSON(doc.Metadata.Title),
		escapeJSON(doc.Metadata.Artist),
		escapeJSON(doc.Metadata.Album),
		escapeJSON(doc.Metadata.By),
		doc.Metadata.Offset,
		escapeJSON(doc.Metadata.Length),
		escapeJSON(doc.Metadata.Language)))

	result.WriteString(`"lines":[`)
	for i, line := range doc.Lines {
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(fmt.Sprintf(`{"time":%d,"timeStr":"%s","text":"%s"}`,
			line.Time, line.TimeStr, escapeJSON(line.Text)))
	}
	result.WriteString("]}")

	return result.String(), nil
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func (doc *LRCDocument) DetectLanguage() string {
	if doc.Metadata.Language != "" {
		lang := strings.ToLower(doc.Metadata.Language)
		switch lang {
		case "en", "eng", "english":
			return "eng"
		case "es", "spa", "spanish":
			return "spa"
		case "fr", "fre", "french":
			return "fre"
		case "de", "ger", "german":
			return "ger"
		case "ja", "jpn", "japanese":
			return "jpn"
		case "ko", "kor", "korean":
			return "kor"
		case "zh", "chi", "chinese":
			return "chi"
		case "ar", "ara", "arabic":
			return "ara"
		case "ur", "urd", "urdu":
			return "urd"
		case "hi", "hin", "hindi":
			return "hin"
		default:
			return lang
		}
	}

	return doc.detectLanguageFromContent()
}

// detectLanguageFromContent uses lingua-go to detect language from lyrics text
func (doc *LRCDocument) detectLanguageFromContent() string {
	if len(doc.Lines) == 0 {
		return "eng"
	}

	var textBuilder strings.Builder
	for _, line := range doc.Lines {
		if strings.TrimSpace(line.Text) != "" {
			textBuilder.WriteString(line.Text)
			textBuilder.WriteString(" ")
		}
	}

	text := strings.TrimSpace(textBuilder.String())
	if text == "" {
		return "eng"
	}

	languages := []lingua.Language{
		lingua.English,
		lingua.Arabic,
		lingua.Urdu,
		lingua.Hindi,
		lingua.Spanish,
		lingua.French,
		lingua.German,
		lingua.Japanese,
		lingua.Korean,
		lingua.Chinese,
		lingua.Portuguese,
		lingua.Italian,
		lingua.Russian,
	}

	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		WithMinimumRelativeDistance(0.9).
		Build()

	if detectedLang, exists := detector.DetectLanguageOf(text); exists {
		// Map lingua.Language to ISO 639-2 codes
		switch detectedLang {
		case lingua.English:
			return "eng"
		case lingua.Arabic:
			return "ara"
		case lingua.Urdu:
			return "urd"
		case lingua.Hindi:
			return "hin"
		case lingua.Spanish:
			return "spa"
		case lingua.French:
			return "fre"
		case lingua.German:
			return "ger"
		case lingua.Japanese:
			return "jpn"
		case lingua.Korean:
			return "kor"
		case lingua.Chinese:
			return "chi"
		case lingua.Portuguese:
			return "por"
		case lingua.Italian:
			return "ita"
		case lingua.Russian:
			return "rus"
		default:
			// If we can't map it, try to get the ISO code
			return strings.ToLower(detectedLang.String()[:3])
		}
	}

	return "eng"
}
