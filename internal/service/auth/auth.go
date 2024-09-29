package auth

import (
	"errors"
	"fmt"
	"time"
	"uni-schedule-backend/internal/apperror"
	"uni-schedule-backend/internal/domain"
	"uni-schedule-backend/internal/repository"
	"uni-schedule-backend/pkg/hash"
)

type JWTManager interface {
	ParseAccessToken(token string) (domain.ID, error)
	ParseRefreshToken(token string) (domain.ID, error)
	GenerateAccessToken(userID domain.ID) (string, error)
	GenerateRefreshToken(userID domain.ID) (string, error)
}

type AuthService struct {
	passwordSalt string
	userRepo     repository.UserRepository
	tokenRepo    repository.TokenRepository
	jwtManager   JWTManager
}

func NewAuthService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository, jwtManager JWTManager, passwordSalt string) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		jwtManager:   jwtManager,
		passwordSalt: passwordSalt,
	}
}

func (s *AuthService) Login(username, password string) (domain.TokenPair, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return domain.TokenPair{}, apperror.ErrInvalidLoginOrPassword
		}
		return domain.TokenPair{}, err
	}

	if !hash.VerifyPassword(password, user.PasswordHash, s.passwordSalt) {
		return domain.TokenPair{}, apperror.ErrInvalidLoginOrPassword
	}

	tokenPair, err := s.generateTokenPair(user.ID)
	if err != nil {
		return domain.TokenPair{}, apperror.NewServiceError("AuthService.Login:", err)
	}

	err = s.tokenRepo.CreateOrUpdate(domain.RefreshToken{
		UserID:       user.ID,
		RefreshToken: tokenPair.RefreshToken,
		UpdatedAt:    time.Now().UTC(),
	})
	if err != nil {
		return domain.TokenPair{}, apperror.NewServiceError("AuthService.Login: create or update token", err)
	}

	return tokenPair, nil
}

func (s *AuthService) Register(username, password string) (domain.TokenPair, error) {
	passwordHash, err := hash.HashPassword(password, s.passwordSalt)
	if err != nil {
		return domain.TokenPair{}, apperror.NewServiceError("AuthService.Register: hashing password", err)
	}

	createdID, err := s.userRepo.Create(domain.UserCreate{
		Username:     username,
		PasswordHash: passwordHash,
		Role:         domain.RoleStudent,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		if errors.Is(err, apperror.ErrAlreadyExists) {
			return domain.TokenPair{}, apperror.ErrUsernameAlreadyTaken
		}
		return domain.TokenPair{}, apperror.NewServiceError("AuthService.Register: create user", err)
	}

	tokenPair, err := s.generateTokenPair(createdID)
	if err != nil {
		return domain.TokenPair{}, apperror.NewServiceError("AuthService.Register:", err)
	}

	err = s.tokenRepo.CreateOrUpdate(domain.RefreshToken{
		UserID:       createdID,
		RefreshToken: tokenPair.RefreshToken,
		UpdatedAt:    time.Now().UTC(),
	})
	if err != nil {
		return domain.TokenPair{}, apperror.NewServiceError("AuthService.Register: create or update token", err)
	}

	return tokenPair, nil
}

func (s *AuthService) RefreshToken(refreshToken string) (domain.TokenPair, error) {
	userID, err := s.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return domain.TokenPair{}, apperror.ErrInvalidRefreshToken
	}

	storedToken, err := s.tokenRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return domain.TokenPair{}, apperror.ErrUserNotFound
		}
		return domain.TokenPair{}, apperror.NewServiceError("AuthService.RefreshToken: getting user by id", err)
	}

	if storedToken.RefreshToken != refreshToken {
		return domain.TokenPair{}, apperror.ErrInvalidRefreshToken
	}

	tokenPair, err := s.generateTokenPair(userID)
	if err != nil {
		return domain.TokenPair{}, apperror.NewServiceError("AuthService.RefreshToken:", err)
	}

	err = s.tokenRepo.CreateOrUpdate(domain.RefreshToken{
		UserID:       userID,
		RefreshToken: tokenPair.RefreshToken,
		UpdatedAt:    time.Now().UTC(),
	})
	if err != nil {
		return domain.TokenPair{}, apperror.NewServiceError("AuthService.RefreshToken: create or update token", err)
	}

	return tokenPair, nil
}

func (s *AuthService) GetUserFromAccessToken(accessToken string) (domain.User, error) {
	userID, err := s.jwtManager.ParseAccessToken(accessToken)
	if err != nil {
		return domain.User{}, apperror.ErrInvalidAccessToken
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return domain.User{}, apperror.ErrUserNotFound
	}

	return user, nil
}

func (s *AuthService) generateTokenPair(userID domain.ID) (domain.TokenPair, error) {
	accessToken, err := s.jwtManager.GenerateAccessToken(userID)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("generateTokenPair.GenerateAccessToken: %w", err)
	}
	refreshToken, err := s.jwtManager.GenerateRefreshToken(userID)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("generateTokenPair.GenerateRefreshToken: %w", err)
	}

	return domain.NewTokenPair(accessToken, refreshToken), nil
}