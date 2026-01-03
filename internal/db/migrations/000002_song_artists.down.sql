ALTER TABLE scan_status DROP COLUMN phase;
DROP INDEX IF EXISTS idx_artists_external_id;
ALTER TABLE artists DROP COLUMN external_id;
DROP INDEX IF EXISTS idx_song_artists_song;
DROP INDEX IF EXISTS idx_song_artists_artist;
DROP TABLE IF EXISTS song_artists;
