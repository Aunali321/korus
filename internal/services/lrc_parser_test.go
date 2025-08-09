package services

import (
	"strings"
	"testing"
)

func TestLRCParser_Parse(t *testing.T) {
	// Sample LRC content
	lrcContent := `[ti:Test Song]
[ar:Test Artist]
[al:Test Album]
[by:Test Creator]
[offset:100]
[length:03:30]
[la:en]

[00:12.34]This is the first line
[00:25.67]This is the second line
[01:15.89]This is the third line
[02:45.12]This is the last line`

	parser := NewLRCParser()
	doc, err := parser.Parse(strings.NewReader(lrcContent))
	
	if err != nil {
		t.Fatalf("Failed to parse LRC: %v", err)
	}

	// Test metadata
	if doc.Metadata.Title != "Test Song" {
		t.Errorf("Expected title 'Test Song', got '%s'", doc.Metadata.Title)
	}
	if doc.Metadata.Artist != "Test Artist" {
		t.Errorf("Expected artist 'Test Artist', got '%s'", doc.Metadata.Artist)
	}
	if doc.Metadata.Album != "Test Album" {
		t.Errorf("Expected album 'Test Album', got '%s'", doc.Metadata.Album)
	}
	if doc.Metadata.By != "Test Creator" {
		t.Errorf("Expected by 'Test Creator', got '%s'", doc.Metadata.By)
	}
	if doc.Metadata.Offset != 100 {
		t.Errorf("Expected offset 100, got %d", doc.Metadata.Offset)
	}
	if doc.Metadata.Length != "03:30" {
		t.Errorf("Expected length '03:30', got '%s'", doc.Metadata.Length)
	}
	if doc.Metadata.Language != "en" {
		t.Errorf("Expected language 'en', got '%s'", doc.Metadata.Language)
	}

	// Test lines
	expectedLines := []struct {
		time int
		text string
	}{
		{12340, "This is the first line"},
		{25670, "This is the second line"},
		{75890, "This is the third line"},
		{165120, "This is the last line"},
	}

	if len(doc.Lines) != len(expectedLines) {
		t.Fatalf("Expected %d lines, got %d", len(expectedLines), len(doc.Lines))
	}

	for i, expected := range expectedLines {
		line := doc.Lines[i]
		if line.Time != expected.time {
			t.Errorf("Line %d: expected time %d, got %d", i, expected.time, line.Time)
		}
		if line.Text != expected.text {
			t.Errorf("Line %d: expected text '%s', got '%s'", i, expected.text, line.Text)
		}
	}
}

func TestLRCParser_DetectLanguage(t *testing.T) {
	tests := []struct {
		language string
		expected string
	}{
		{"en", "eng"},
		{"english", "eng"},
		{"es", "spa"},
		{"spanish", "spa"},
		{"fr", "fre"},
		{"french", "fre"},
		{"de", "ger"},
		{"german", "ger"},
		{"", "eng"}, // default
	}

	for _, test := range tests {
		doc := &LRCDocument{
			Metadata: LRCMetadata{Language: test.language},
		}
		result := doc.DetectLanguage()
		if result != test.expected {
			t.Errorf("Language '%s': expected '%s', got '%s'", test.language, test.expected, result)
		}
	}
}

func TestLRCParser_ToJSON(t *testing.T) {
	doc := &LRCDocument{
		Metadata: LRCMetadata{
			Title:  "Test Song",
			Artist: "Test Artist",
		},
		Lines: []LRCLine{
			{Time: 12340, TimeStr: "[00:12.34]", Text: "First line"},
			{Time: 25670, TimeStr: "[00:25.67]", Text: "Second line"},
		},
	}

	json, err := doc.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	// Check that JSON contains expected elements
	expectedElements := []string{
		`"title":"Test Song"`,
		`"artist":"Test Artist"`,
		`"time":12340`,
		`"text":"First line"`,
		`"time":25670`,
		`"text":"Second line"`,
	}

	for _, element := range expectedElements {
		if !strings.Contains(json, element) {
			t.Errorf("JSON missing expected element: %s\nGenerated JSON: %s", element, json)
		}
	}
}