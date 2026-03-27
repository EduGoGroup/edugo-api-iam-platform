package service

import (
	"context"
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

// GenerateRefreshJWT generates a refresh token embedding schoolID so the backend
// can reconstruct the correct school context on token rotation without losing the
// school the user selected (including via switchContext).
func (s *TokenService) GenerateRefreshJWT(userID, email, schoolID string) (string, int64, error) {
	token, expiresAt, err := s.jwtManager.GenerateMinimalToken(userID, email, schoolID, s.refreshDuration)
	if err != nil {
		return "", 0, err
	}
	return token, int64(time.Until(expiresAt).Seconds()), nil
}

// ValidateRefreshJWT validates a refresh token JWT and returns userID, email and schoolID.
// schoolID may be empty for legacy tokens or users without an active school context.
func (s *TokenService) ValidateRefreshJWT(token string) (string, string, string, error) {
	claims, err := s.jwtManager.ValidateMinimalToken(token)
	if err != nil {
		return "", "", "", err
	}
	return claims.UserID, claims.Email, claims.SchoolID, nil
}

// GenerateTokenPairWithContext generates an access+refresh token pair with RBAC context.
// The schoolID from activeContext is embedded in the refresh token so context is preserved
// across token rotations.
func (s *TokenService) GenerateTokenPairWithContext(userID, email string, activeContext *auth.UserContext) (*dto.LoginResponse, error) {
	accessToken, expiresAt, err := s.jwtManager.GenerateTokenWithContext(userID, email, activeContext, s.accessDuration)
	if err != nil {
		return nil, fmt.Errorf("error generating access token: %w", err)
	}

	refreshJWT, _, err := s.GenerateRefreshJWT(userID, email, activeContext.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("error generating refresh token: %w", err)
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshJWT,
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

// ValidateAccessToken validates an access token and returns the raw claims.
// Used internally (e.g., for extracting JTI during logout).
func (s *TokenService) ValidateAccessToken(token string) (*auth.Claims, error) {
	return s.jwtManager.ValidateToken(token)
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
