package ai

import (
	"context"
)

type Conversation struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (s *Service) CreateConversation(ctx context.Context, userID int64, title string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO ai_conversations(user_id, title) VALUES(?, ?)`, userID, title)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Service) ConversationOwner(ctx context.Context, id int64) (int64, error) {
	var uid int64
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM ai_conversations WHERE id = ?`, id).Scan(&uid)
	return uid, err
}

func (s *Service) AppendMessage(ctx context.Context, convID int64, role, content string) error {
	if _, err := s.db.ExecContext(ctx, `INSERT INTO ai_messages(conversation_id, role, content) VALUES(?, ?, ?)`, convID, role, content); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE ai_conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, convID)
	return err
}

// maxHistoryMessages bounds how much of a conversation is replayed into the
// model's context; older turns stay stored but are not sent.
const maxHistoryMessages = 40

func (s *Service) LoadHistory(ctx context.Context, convID int64) ([]ChatMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT role, content FROM ai_messages WHERE conversation_id = ? ORDER BY id DESC LIMIT ?`, convID, maxHistoryMessages)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ChatMessage, 0)
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.Role, &m.Text); err != nil {
			continue
		}
		out = append(out, m)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

func (s *Service) ListConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, created_at, updated_at FROM ai_conversations WHERE user_id = ? ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Conversation, 0)
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) DeleteConversation(ctx context.Context, id, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_conversations WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

// CachedWrapped returns the cached Wrapped HTML body for a period, if any.
func (s *Service) CachedWrapped(ctx context.Context, userID int64, periodType, periodKey string) (string, bool) {
	var html string
	err := s.db.QueryRowContext(ctx, `SELECT spec FROM ai_wrapped WHERE user_id = ? AND period_type = ? AND period_key = ?`, userID, periodType, periodKey).Scan(&html)
	if err != nil || html == "" {
		return "", false
	}
	return html, true
}

// SaveWrapped stores (overwriting) the generated Wrapped HTML body for a period.
func (s *Service) SaveWrapped(ctx context.Context, userID int64, periodType, periodKey, html string) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO ai_wrapped(user_id, period_type, period_key, spec) VALUES(?, ?, ?, ?)`, userID, periodType, periodKey, html)
	return err
}
