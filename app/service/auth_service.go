package service

import (
	"context"
	"errors"
	"latihan-sintaks/app/model"
	"latihan-sintaks/app/repository"
	"latihan-sintaks/helper"
)

type AuthService interface {
	Register(ctx context.Context, req model.RegisterRequest) (*model.UserResponse, error)
	Login(ctx context.Context, req model.LoginRequest) (*model.TokenPair, error) // <- BERUBAH
	Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error)  // <- BERUBAH
	Logout(ctx context.Context, refreshToken string) error
	GetMe(ctx context.Context, username string) (*model.UserResponse, error)
}

type authService struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
}

func NewAuthService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository) AuthService {
	return &authService{userRepo: userRepo, tokenRepo: tokenRepo}
}

func (s *authService) Register(ctx context.Context, req model.RegisterRequest) (*model.UserResponse, error) {
	if err := ValidateRegisterInput(req.Username, req.Email, req.Password); err != nil {
		return nil, err
	}

	hashedPassword, err := helper.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("gagal mengenkripsi password")
	}

	user := &model.UserResponse{
		Username: req.Username,
		Email:    req.Email,
	}

	if err := s.userRepo.Create(ctx, user, hashedPassword); err != nil {
		return nil, errors.New("gagal menyimpan user atau username sudah terdaftar")
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, req model.LoginRequest) (*model.TokenPair, error) {
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, errors.New("username atau password salah")
	}

	if !helper.VerifyPassword(user.Password, req.Password) {
		return nil, errors.New("username atau password salah")
	}

	accessToken, err := helper.GenerateAccessToken(user.Username, user.Role)
	if err != nil {
		return nil, errors.New("gagal membuat access token")
	}

	refreshToken, err := helper.GenerateRefreshToken()
	if err != nil {
		return nil, errors.New("gagal membuat refresh token")
	}

	if err := s.tokenRepo.Save(ctx, user.ID, refreshToken); err != nil {
		return nil, errors.New("gagal menyimpan refresh token")
	}

	// <- BERUBAH: Menggunakan model.TokenPair
	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	userID, err := s.tokenRepo.FindByToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.New("refresh token tidak valid")
	}

	_ = s.tokenRepo.Delete(ctx, refreshToken)

	newAccessToken, _ := helper.GenerateAccessToken("user", "user")
	newRefreshToken, _ := helper.GenerateRefreshToken()

	_ = s.tokenRepo.Save(ctx, userID, newRefreshToken)

	// <- BERUBAH: Menggunakan model.TokenPair
	return &model.TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	return s.tokenRepo.Delete(ctx, refreshToken)
}

func (s *authService) GetMe(ctx context.Context, username string) (*model.UserResponse, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}

	return &model.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}