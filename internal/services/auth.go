package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Aunali321/korus/internal/models"
)

type AuthService struct {
	db         *sql.DB
	jwtSecret  []byte
	tokenTTL   time.Duration
	refreshTTL time.Duration
}

func NewAuthService(db *sql.DB, secret []byte, tokenTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{db: db, jwtSecret: secret, tokenTTL: tokenTTL, refreshTTL: refreshTTL}
}

type Tokens struct {
	Access  string
	Refresh string
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) (models.User, Tokens, error) {
	if username == "" || email == "" || password == "" {
		return models.User{}, Tokens{}, errors.New("missing credentials")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, Tokens{}, fmt.Errorf("hash password: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, email, role)
		VALUES (?, ?, ?, 'user')
	`, username, string(hash), email)
	if err != nil {
		return models.User{}, Tokens{}, fmt.Errorf("insert user: %w", err)
	}
	id, _ := res.LastInsertId()
	user := models.User{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         "user",
		CreatedAt:    time.Now(),
	}
	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		return models.User{}, Tokens{}, err
	}
	return user, tokens, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (models.User, Tokens, error) {
	var user models.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, email, role, created_at
		FROM users
		WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, Tokens{}, errors.New("invalid credentials")
		}
		return models.User{}, Tokens{}, fmt.Errorf("query user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return models.User{}, Tokens{}, errors.New("invalid credentials")
	}
	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		return models.User{}, Tokens{}, err
	}
	return user, tokens, nil
}

func (s *AuthService) issueTokens(ctx context.Context, user models.User) (Tokens, error) {
	sessionID := uuid.NewString()
	expiresAt := time.Now().Add(s.tokenTTL)
	claims := jwt.MapClaims{
		"sub": fmt.Sprintf("%d", user.ID),
		"sid": sessionID,
		"rol": user.Role,
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	access, err := tok.SignedString(s.jwtSecret)
	if err != nil {
		return Tokens{}, fmt.Errorf("sign token: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)
	`, sessionID, user.ID, expiresAt); err != nil {
		return Tokens{}, fmt.Errorf("store session: %w", err)
	}
	refresh, err := s.createRefreshToken(ctx, user.ID, sessionID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{Access: access, Refresh: refresh}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenStr string) (models.User, error) {
	if tokenStr == "" {
		return models.User{}, errors.New("missing token")
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return models.User{}, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return models.User{}, errors.New("invalid token claims")
	}
	sid, _ := claims["sid"].(string)
	uidStr, _ := claims["sub"].(string)
	var user models.User
	var expires time.Time
	err = s.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.password_hash, u.email, u.role, u.created_at, s.expires_at
		FROM users u
		JOIN sessions s ON s.user_id = u.id
		WHERE u.id = ? AND s.token = ?
	`, uidStr, sid).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.Role, &user.CreatedAt, &expires)
	if err != nil {
		return models.User{}, errors.New("session not found")
	}
	if time.Now().After(expires) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, sid)
		return models.User{}, errors.New("session expired")
	}
	return user, nil
}

func (s *AuthService) Logout(ctx context.Context, tokenStr string) error {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return errors.New("invalid token")
	}
	claims, _ := token.Claims.(jwt.MapClaims)
	sid, _ := claims["sid"].(string)
	if sid == "" {
		return errors.New("missing session id")
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, sid)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE session_token = ?`, sid)
	return err
}

func (s *AuthService) CleanupSessions(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup sessions: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (models.User, Tokens, error) {
	if refreshToken == "" {
		return models.User{}, Tokens{}, errors.New("missing refresh token")
	}
	hash := hashToken(refreshToken)
	var user models.User
	var sessionToken string
	var expires time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT rt.session_token, rt.expires_at, u.id, u.username, u.password_hash, u.email, u.role, u.created_at
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		WHERE rt.token_hash = ? AND rt.revoked = 0
	`, hash).Scan(&sessionToken, &expires, &user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.Role, &user.CreatedAt)
	if err != nil {
		return models.User{}, Tokens{}, errors.New("invalid refresh token")
	}
	if time.Now().After(expires) {
		_, _ = s.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked = 1 WHERE token_hash = ?`, hash)
		return models.User{}, Tokens{}, errors.New("refresh expired")
	}
	// rotate: revoke old token and issue new access + refresh
	if _, err := s.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked = 1 WHERE token_hash = ?`, hash); err != nil {
		return models.User{}, Tokens{}, fmt.Errorf("revoke refresh: %w", err)
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, sessionToken)
	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		return models.User{}, Tokens{}, err
	}
	return user, tokens, nil
}

func (s *AuthService) createRefreshToken(ctx context.Context, userID int64, sessionToken string) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("refresh token: %w", err)
	}
	token := fmt.Sprintf("%x", tokenBytes)
	hash := hashToken(token)
	expires := time.Now().Add(s.refreshTTL)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (token_hash, user_id, session_token, expires_at)
		VALUES (?, ?, ?, ?)
	`, hash, userID, sessionToken, expires)
	if err != nil {
		return "", fmt.Errorf("store refresh: %w", err)
	}
	return token, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", sum[:])
}
