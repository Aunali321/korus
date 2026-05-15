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

-- Bump schema_version to invalidate any cached schema in connections that
-- saw the old NOT NULL definition. Without this, the active Go connection
-- continues to enforce the old constraint even after the schema text
-- changes, causing post-migration UPDATEs to NULL to fail spuriously.
PRAGMA schema_version = (SELECT schema_version + 1 FROM pragma_schema_version);

PRAGMA integrity_check;
