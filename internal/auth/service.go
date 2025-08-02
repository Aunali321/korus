package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"korus/internal/database"
	"korus/internal/models"
)

type Service struct {
	db           *database.DB
	tokenManager *TokenManager
}

func NewService(db *database.DB, tokenManager *TokenManager) *Service {
	return &Service{
		db:           db,
		tokenManager: tokenManager,
	}
}

func (s *Service) Login(ctx context.Context, username, password string) (*TokenPair, *models.User, error) {
	// Get user by username
	user, err := s.getUserByUsername(ctx, username)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, fmt.Errorf("invalid credentials")
		}
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if !VerifyPassword(user.PasswordHash, password) {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	// Generate token pair
	tokens, err := s.tokenManager.GenerateTokenPair(user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store refresh token
	err = s.storeRefreshToken(ctx, user.ID, tokens.RefreshToken, s.tokenManager.GetRefreshTokenExpiry())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Update last login
	err = s.updateLastLogin(ctx, user.ID)
	if err != nil {
		// Log error but don't fail the login
		fmt.Printf("Warning: failed to update last login for user %d: %v\n", user.ID, err)
	}

	return tokens, user, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	session, err := s.getSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if token is expired
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired token
		s.deleteSession(ctx, session.ID)
		return nil, fmt.Errorf("refresh token expired")
	}

	// Get user
	user, err := s.getUserByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate new access token only
	accessToken, expiresAt, err := s.tokenManager.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // Keep same refresh token
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.deleteSessionByRefreshToken(ctx, refreshToken)
}

func (s *Service) CreateUser(ctx context.Context, username, email, password string) (*models.User, error) {
	// Validate password strength
	if err := ValidatePasswordStrength(password); err != nil {
		return nil, err
	}

	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	query := `
		INSERT INTO users (username, email, password_hash, role) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id, username, email, role, created_at
	`

	var user models.User
	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}

	err = s.db.QueryRowContext(ctx, query, username, emailPtr, hashedPassword, "user").
		Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user.Email = emailPtr
	return &user, nil
}

func (s *Service) CreateAdminUser(ctx context.Context, username, password string) (*models.User, error) {
	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create admin user
	query := `
		INSERT INTO users (username, password_hash, role) 
		VALUES ($1, $2, 'admin') 
		RETURNING id, username, email, role, created_at
	`

	var user models.User
	err = s.db.QueryRowContext(ctx, query, username, hashedPassword).
		Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	return &user, nil
}

func (s *Service) GetUserByID(ctx context.Context, userID int) (*models.User, error) {
	return s.getUserByID(ctx, userID)
}

func (s *Service) ValidateToken(ctx context.Context, tokenString string) (*models.User, error) {
	claims, err := s.tokenManager.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Get fresh user data to ensure user still exists and is active
	user, err := s.getUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

func (s *Service) HasUsers(ctx context.Context) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to count users: %w", err)
	}
	return count > 0, nil
}

// Private helper methods

func (s *Service) getUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, last_login 
		FROM users 
		WHERE username = $1
	`

	var user models.User
	err := s.db.QueryRowContext(ctx, query, username).
		Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, 
			 &user.Role, &user.CreatedAt, &user.LastLogin)
	
	return &user, err
}

func (s *Service) getUserByID(ctx context.Context, userID int) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, last_login 
		FROM users 
		WHERE id = $1
	`

	var user models.User
	err := s.db.QueryRowContext(ctx, query, userID).
		Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, 
			 &user.Role, &user.CreatedAt, &user.LastLogin)
	
	return &user, err
}

func (s *Service) storeRefreshToken(ctx context.Context, userID int, refreshToken string, expiresAt time.Time) error {
	query := `
		INSERT INTO user_sessions (user_id, refresh_token, expires_at) 
		VALUES ($1, $2, $3)
	`
	_, err := s.db.ExecContext(ctx, query, userID, refreshToken, expiresAt)
	return err
}

func (s *Service) getSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.UserSession, error) {
	query := `
		SELECT id, user_id, refresh_token, expires_at, created_at 
		FROM user_sessions 
		WHERE refresh_token = $1
	`

	var session models.UserSession
	err := s.db.QueryRowContext(ctx, query, refreshToken).
		Scan(&session.ID, &session.UserID, &session.RefreshToken, 
			 &session.ExpiresAt, &session.CreatedAt)
	
	return &session, err
}

func (s *Service) deleteSession(ctx context.Context, sessionID int) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM user_sessions WHERE id = $1", sessionID)
	return err
}

func (s *Service) deleteSessionByRefreshToken(ctx context.Context, refreshToken string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM user_sessions WHERE refresh_token = $1", refreshToken)
	return err
}

func (s *Service) updateLastLogin(ctx context.Context, userID int) error {
	_, err := s.db.ExecContext(ctx, "UPDATE users SET last_login = NOW() WHERE id = $1", userID)
	return err
}