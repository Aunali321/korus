-- Many-to-many relationship between songs and artists
CREATE TABLE IF NOT EXISTS song_artists (
    song_id INTEGER NOT NULL,
    artist_id INTEGER NOT NULL,
    role TEXT NOT NULL DEFAULT 'primary',
    position INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (song_id, artist_id),
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
    FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_song_artists_artist ON song_artists(artist_id);
CREATE INDEX IF NOT EXISTS idx_song_artists_song ON song_artists(song_id);

-- External ID for deduplication (from metadata enrichment)
ALTER TABLE artists ADD COLUMN external_id TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_artists_external_id ON artists(external_id) WHERE external_id IS NOT NULL;

-- Scan phase tracking
ALTER TABLE scan_status ADD COLUMN phase TEXT NOT NULL DEFAULT 'scanning';
