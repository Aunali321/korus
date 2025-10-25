-- Description: Create song_embeddings table for recommendation embeddings

CREATE TABLE IF NOT EXISTS song_embeddings (
    song_id INTEGER PRIMARY KEY REFERENCES songs(id) ON DELETE CASCADE,
    embedding REAL[] NOT NULL,
    dim SMALLINT NOT NULL,
    segments SMALLINT NOT NULL DEFAULT 0,
    method TEXT NOT NULL,
    model TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_song_embeddings_model ON song_embeddings(model);
