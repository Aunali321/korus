# Korus Lyrics Feature - Implementation Guide

## Overview

Korus now supports comprehensive lyrics functionality with automatic extraction from multiple sources and seamless API integration. This document provides complete implementation details for client developers.

## Feature Summary

- **Multi-source lyrics extraction**: Embedded ID3 tags, external .lrc files, external .txt files
- **Synchronized lyrics support**: Custom LRC parser with precise timing data
- **Multi-language support**: ISO 639-2 language codes with automatic detection
- **Automatic integration**: Lyrics are always included in song responses
- **No additional API calls needed**: All lyrics for all languages included in `/api/songs/{id}`

## API Integration

### Song Response Structure

All song endpoints now include a `lyrics` array with complete lyrics data:

**Endpoint**: `GET /api/songs/{id}`

**Response Structure**:
```json
{
  "id": 1,
  "title": "Song Title",
  "album_id": 1,
  "artist_id": 1,
  "duration": 240,
  // ... other song fields ...
  "lyrics": [
    {
      "id": 1,
      "song_id": 1,
      "content": "Plain text lyrics content...",
      "type": "unsynced",
      "source": "embedded",
      "language": "eng",
      "created_at": "2025-08-09T12:00:00Z"
    },
    {
      "id": 2,
      "song_id": 1,
      "content": "{\"metadata\":{\"title\":\"Song Title\",\"artist\":\"Artist Name\",\"language\":\"eng\"},\"lines\":[{\"time\":1230,\"timeStr\":\"[00:01.23]\",\"text\":\"First line\"},{\"time\":5670,\"timeStr\":\"[00:05.67]\",\"text\":\"Second line\"}]}",
      "type": "synced",
      "source": "external_lrc",
      "language": "eng",
      "created_at": "2025-08-09T12:00:00Z"
    },
    {
      "id": 3,
      "song_id": 1,
      "content": "Letra en español...",
      "type": "unsynced",
      "source": "external_txt",
      "language": "spa",
      "created_at": "2025-08-09T12:00:00Z"
    }
  ]
}
```

## Lyrics Data Structure

### Lyrics Object Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Unique lyrics entry ID |
| `song_id` | integer | Associated song ID |
| `content` | string | Lyrics content (plain text or JSON for synced) |
| `type` | string | `"synced"` or `"unsynced"` |
| `source` | string | `"embedded"`, `"external_lrc"`, or `"external_txt"` |
| `language` | string | ISO 639-2 language code |
| `created_at` | string | ISO 8601 timestamp |

### Lyrics Types

#### 1. Unsynced Lyrics (`type: "unsynced"`)
- **Content**: Plain text lyrics
- **Sources**: Embedded ID3 tags, .txt files
- **Usage**: Display as static text

#### 2. Synchronized Lyrics (`type: "synced"`)  
- **Content**: JSON string with metadata and timed lines
- **Sources**: .lrc files
- **Usage**: Parse JSON for karaoke-style playback

## Synchronized Lyrics JSON Format

For `type: "synced"`, the `content` field contains a JSON string with this structure:

```json
{
  "metadata": {
    "title": "Song Title",
    "artist": "Artist Name",
    "album": "Album Name",
    "by": "Creator Name",
    "offset": 100,
    "length": "03:30",
    "language": "eng"
  },
  "lines": [
    {
      "time": 1230,
      "timeStr": "[00:01.23]",
      "text": "First lyrics line"
    },
    {
      "time": 5670,
      "timeStr": "[00:05.67]",
      "text": "Second lyrics line"
    }
  ]
}
```

### Synchronized Lyrics Fields

| Field | Type | Description |
|-------|------|-------------|
| `metadata.title` | string | Song title from LRC file |
| `metadata.artist` | string | Artist name from LRC file |
| `metadata.offset` | integer | Timing offset in milliseconds |
| `lines[].time` | integer | Line timestamp in milliseconds |
| `lines[].timeStr` | string | Original LRC timestamp format |
| `lines[].text` | string | Lyrics text for this timestamp |

## Language Support

### Language Codes (ISO 639-2)
- `eng` - English
- `ara` - Arabic
- `urd` - Urdu
- `hin` - Hindi
- `spa` - Spanish  
- `fre` - French
- `ger` - German
- `jpn` - Japanese
- `kor` - Korean
- `chi` - Chinese
- `por` - Portuguese
- `ita` - Italian
- `rus` - Russian

### Language Detection
- **LRC files**: Parsed from `[la:language]` metadata tags when available
- **Content analysis**: Uses lingua-go library for automatic language detection from lyrics text
- **Auto-metadata filling**: Missing LRC metadata (title, artist, album) automatically populated from song information
- **Multiple languages**: Songs can have lyrics in multiple languages

## Client Implementation Examples

### Display All Lyrics
```javascript
// Get song with lyrics
const response = await fetch(`/api/songs/${songId}`);
const song = await response.json();

// Display all available lyrics
song.lyrics.forEach(lyric => {
    console.log(`${lyric.language} (${lyric.type}):`, lyric.content);
});
```

### Handle Synchronized Lyrics
```javascript
// Find synchronized lyrics
const syncedLyrics = song.lyrics.find(l => l.type === 'synced');
if (syncedLyrics) {
    const lrcData = JSON.parse(syncedLyrics.content);
    
    // Use for karaoke-style display
    lrcData.lines.forEach(line => {
        setTimeout(() => {
            displayLyricsLine(line.text);
        }, line.time + lrcData.metadata.offset);
    });
}
```

### Language Selection
```javascript
// Get lyrics in preferred language
function getLyricsByLanguage(lyrics, preferredLang = 'eng') {
    return lyrics.find(l => l.language === preferredLang) || lyrics[0];
}

const preferredLyrics = getLyricsByLanguage(song.lyrics, 'spa');
```

### Priority-Based Display
```javascript
// Display lyrics by priority: synced > unsynced
function getPreferredLyrics(lyrics, language = 'eng') {
    const langLyrics = lyrics.filter(l => l.language === language);
    return langLyrics.find(l => l.type === 'synced') || 
           langLyrics.find(l => l.type === 'unsynced');
}
```

## File System Integration

### Supported File Formats
Korus automatically scans for lyrics files alongside audio files:

#### External LRC Files (Synchronized)
- **Pattern**: `{audiofile}.lrc` (e.g., `song.mp3` → `song.lrc`)
- **Format**: Standard LRC format with timestamps
- **Example**:
```lrc
[ti:Song Title]
[ar:Artist Name]
[la:eng]
[00:12.34]First line of lyrics
[00:25.67]Second line of lyrics
```

#### External TXT Files (Unsynchronized)
- **Pattern**: `{audiofile}.txt` (e.g., `song.mp3` → `song.txt`)
- **Format**: Plain text
- **Example**:
```
First line of lyrics
Second line of lyrics
Chorus repeated here
```

#### Embedded Lyrics
- **Source**: ID3v2 USLT frames in audio files
- **Format**: Plain text extracted automatically
- **Supported formats**: MP3, M4A, FLAC, OGG

## Database Schema

### Lyrics Table Structure
```sql
CREATE TABLE lyrics (
    id SERIAL PRIMARY KEY,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'unsynced',
    source VARCHAR(50) NOT NULL,
    language VARCHAR(3) DEFAULT 'eng',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(song_id, language, type)
);
```

## Migration and Compatibility

### Automatic Processing
- **New files**: Lyrics automatically extracted during library scan
- **Existing files**: Re-scan library to extract lyrics from existing songs
- **No breaking changes**: Lyrics are optional and won't affect existing functionality

### API Compatibility
- **Backward compatible**: Existing clients will receive lyrics but can ignore the field
- **No additional endpoints**: All lyrics data available through existing song endpoints
- **Empty arrays**: Songs without lyrics return `"lyrics": []`

## Performance Considerations

### Database Queries
- **Efficient loading**: Batch lyrics loading for multiple songs
- **Indexed queries**: Optimized with `song_id`, `type`, and `source` indexes
- **Automatic inclusion**: No additional API calls needed

### Storage Efficiency
- **Unique constraints**: Prevents duplicate lyrics entries
- **Cascade deletion**: Lyrics automatically deleted when songs are removed
- **Language separation**: Multiple lyrics per song for different languages

## Implementation Checklist

### Server-Side (Handled Automatically)
- [x] Lyrics extraction during file scanning
- [x] Database storage with proper indexing  
- [x] API integration in song responses
- [x] Multi-language support
- [x] Synchronized lyrics parsing

### Client-Side Implementation
- [ ] Parse lyrics from song API responses
- [ ] Handle both synchronized and unsynchronized lyrics
- [ ] Implement language selection UI
- [ ] Add karaoke-style synchronized display
- [ ] Graceful handling of songs without lyrics

## Troubleshooting

### Common Issues

#### No Lyrics in API Response
- **Check**: Ensure lyrics files (.lrc/.txt) exist alongside audio files
- **Check**: Verify audio files have embedded lyrics (ID3 USLT tags)
- **Solution**: Re-scan library after adding lyrics files

#### Synchronized Lyrics Not Working
- **Check**: Verify .lrc file format is valid
- **Check**: Parse JSON content for synchronized lyrics
- **Solution**: Use LRC validator tools to verify file format

#### Wrong Language Detection  
- **Check**: Add `[la:language]` tag to .lrc files
- **Solution**: Use proper ISO 639-2 language codes

#### Multiple Lyrics Entries
- **Expected**: Songs can have multiple lyrics in different languages/formats
- **Solution**: Implement client-side filtering by language preference

## Support

For issues or questions about the lyrics feature implementation, check:

1. **API Documentation**: `API.md` - Complete endpoint documentation
2. **System Design**: `DESIGN.md` - Technical implementation details  
3. **Example Files**: Include sample .lrc and .txt files with your music
4. **Testing**: Use the provided song examples to verify implementation

The lyrics feature is fully integrated and requires no additional server configuration - simply ensure your music files have accompanying lyrics files or embedded lyrics data.