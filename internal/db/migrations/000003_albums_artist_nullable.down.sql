-- Roll back to NOT NULL artist_id and ON DELETE CASCADE. Lossy: any
-- compilation albums (artist_id IS NULL) need a real artist to satisfy
-- NOT NULL — repoint them at the lowest existing artist id before
-- rewriting the schema.
UPDATE albums
SET artist_id = (SELECT MIN(id) FROM artists)
WHERE artist_id IS NULL;

PRAGMA writable_schema = ON;

UPDATE sqlite_master
SET sql = 'CREATE TABLE albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    artist_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    year INTEGER,
    cover_path TEXT,
    mbid TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE CASCADE
)'
WHERE type = 'table' AND name = 'albums';

PRAGMA writable_schema = OFF;

PRAGMA integrity_check;
