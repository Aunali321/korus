CREATE TABLE ai_wrapped (
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_type TEXT NOT NULL,
    period_key  TEXT NOT NULL,
    spec        TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, period_type, period_key)
);
