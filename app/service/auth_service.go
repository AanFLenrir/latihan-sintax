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
	Login(ctx context.Context, req model.LoginRequest) (*model.TokenResponse, error)
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

	err = s.userRepo.Create(ctx, user, hashedPassword)
	if err != nil {
		return nil, errors.New("username atau email sudah terdaftar")
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, req model.LoginRequest) (*model.TokenResponse, error) {
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		helper.VerifyDummyPassword(req.Password)
		return nil, errors.New("username atau password salah")
	}

	if !helper.CheckPasswordHash(req.Password, user.Password) {
		return nil, errors.New("username atau password salah")
	}

	accessToken, err := helper.GenerateAccessToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, errors.New("gagal membuat access token")
	}

	refreshToken := helper.GenerateRandomString(32)
	err = s.tokenRepo.Save(ctx, user.ID, refreshToken)
	if err != nil {
		return nil, errors.New("gagal menyimpan refresh token")
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
	}, nil
}