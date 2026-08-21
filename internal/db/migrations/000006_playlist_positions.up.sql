-- Playlist positions were written by three schemes: unix timestamps from the
-- HTTP add endpoint, 1-based indices from reorder, and max+1 from the AI
-- tools. Mixed in one playlist, the timestamped rows sorted permanently last.
-- Collapse every playlist to dense zero-based positions, preserving the order
-- the rows currently sort in.
UPDATE playlist_songs
SET position = (
    SELECT COUNT(*)
    FROM playlist_songs other
    WHERE other.playlist_id = playlist_songs.playlist_id
      AND (other.position < playlist_songs.position
           OR (other.position = playlist_songs.position AND other.song_id < playlist_songs.song_id))
);
