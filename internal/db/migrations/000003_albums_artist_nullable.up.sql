-- albums.artist_id becomes nullable. NULL means "compilation / multiple
-- distinct primary artists across the album's songs". The album's actual
-- per-track artists live in song_artists, which is the source of truth.
-- FK changes from CASCADE to SET NULL so deleting an artist downgrades
-- their album to a compilation rather than destroying it.
--
-- Implementation note: the standard SQLite table-rebuild pattern (CREATE
-- new, INSERT SELECT, DROP old, RENAME) is UNSAFE here because golang-migrate
-- wraps each migration in a transaction, and PRAGMA foreign_keys = OFF is a
-- no-op inside a transaction. DROP TABLE albums would cascade through
-- songs.album_id (ON DELETE CASCADE) and obliterate play_history. We use
-- writable_schema to rewrite the table definition in place — the rows are
-- not touched, so no FK cascade fires.
PRAGMA writable_schema = ON;

UPDATE sqlite_master
SET sql = 'CREATE TABLE albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    artist_id INTEGER,
    title TEXT NOT NULL,
    year INTEGER,
    cover_path TEXT,
    mbid TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE SET NULL
)'
WHERE type = 'table' AND name = 'albums';

PRAGMA writable_schema = OFF;

PRAGMA integrity_check;
