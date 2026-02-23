package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/dto"
	"github.com/EduGoGroup/edugo-shared/auth"
)

// TokenService manages JWT token operations
type TokenService struct {
	jwtManager      *auth.JWTManager
	accessDuration  time.Duration
	refreshDuration time.Duration
}

// NewTokenService creates a new TokenService
func NewTokenService(jwtManager *auth.JWTManager, accessDuration, refreshDuration time.Duration) *TokenService {
	if accessDuration == 0 {
		accessDuration = 15 * time.Minute
	}
	if refreshDuration == 0 {
		refreshDuration = 7 * 24 * time.Hour
	}
	return &TokenService{
		jwtManager:      jwtManager,
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
	}
}

// GenerateTokenPairWithContext generates an access+refresh token pair with RBAC context
func (s *TokenService) GenerateTokenPairWithContext(userID, email string, activeContext *auth.UserContext) (*dto.LoginResponse, error) {
	accessToken, expiresAt, err := s.jwtManager.GenerateTokenWithContext(userID, email, activeContext, s.accessDuration)
	if err != nil {
		return nil, fmt.Errorf("error generating access token: %w", err)
	}

	refreshToken, err := auth.GenerateRefreshToken(s.refreshDuration)
	if err != nil {
		return nil, fmt.Errorf("error generating refresh token: %w", err)
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    int64(time.Until(expiresAt).Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// GenerateAccessTokenWithContext generates only a new access token with RBAC context
func (s *TokenService) GenerateAccessTokenWithContext(userID, email string, activeContext *auth.UserContext) (*dto.RefreshResponse, error) {
	accessToken, expiresAt, err := s.jwtManager.GenerateTokenWithContext(userID, email, activeContext, s.accessDuration)
	if err != nil {
		return nil, fmt.Errorf("error generating access token: %w", err)
	}

	return &dto.RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(time.Until(expiresAt).Seconds()),
		TokenType:   "Bearer",
	}, nil
}

// VerifyToken validates a JWT token and returns token info
func (s *TokenService) VerifyToken(_ context.Context, token string) (*dto.VerifyTokenResponse, error) {
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return &dto.VerifyTokenResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	expiresAt := claims.ExpiresAt.Time
	schoolID := ""
	if claims.ActiveContext != nil {
		schoolID = claims.ActiveContext.SchoolID
	}

	return &dto.VerifyTokenResponse{
		Valid:     true,
		UserID:    claims.UserID,
		Email:     claims.Email,
		SchoolID:  schoolID,
		ExpiresAt: &expiresAt,
	}, nil
}

// hashToken generates a SHA256 hash of a token for cache keys
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return "auth:token:" + hex.EncodeToString(hash[:])
}
