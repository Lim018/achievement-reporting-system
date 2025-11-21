package service

import (
	"database/sql"
	"errors"
	"go-fiber/app/model"
	"go-fiber/app/repository"
	"go-fiber/utils"

	// "github.com/gofiber/fiber/v2"
)

func LoginService(db *sql.DB, req model.LoginRequest) (*model.LoginResponse, error) {
	user, passwordHash, err := repository.FindUserByUsernameOrEmail(db, req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("username atau email tidak ditemukan")
		}
		return nil, errors.New("terjadi kesalahan saat login")
	}

	if !user.IsActive {
		return nil, errors.New("akun tidak aktif")
	}

	if !utils.CheckPassword(req.Password, passwordHash) {
		return nil, errors.New("password salah")
	}

	token, err := utils.GenerateToken(*user)
	if err != nil {
		return nil, errors.New("gagal generate token")
	}

	refreshToken, expiresAt, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, errors.New("gagal generate refresh token")
	}

	err = repository.SaveRefreshToken(db, user.ID, refreshToken, expiresAt)
	if err != nil {
		return nil, errors.New("gagal menyimpan refresh token")
	}

	return &model.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user.ToUserResponse(),
	}, nil
}

func RefreshTokenService(db *sql.DB, req model.RefreshTokenRequest) (*model.LoginResponse, error) {
	claims, err := utils.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("refresh token tidak valid")
	}

	rt, err := repository.FindRefreshToken(db, req.RefreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("refresh token tidak ditemukan")
		}
		return nil, errors.New("terjadi kesalahan")
	}

	if rt.UserID != claims.UserID {
		return nil, errors.New("refresh token tidak valid")
	}

	user, err := repository.FindUserByID(db, claims.UserID)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}

	token, err := utils.GenerateToken(*user)
	if err != nil {
		return nil, errors.New("gagal generate token")
	}

	newRefreshToken, expiresAt, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, errors.New("gagal generate refresh token")
	}

	repository.DeleteRefreshToken(db, req.RefreshToken)

	err = repository.SaveRefreshToken(db, user.ID, newRefreshToken, expiresAt)
	if err != nil {
		return nil, errors.New("gagal menyimpan refresh token")
	}

	return &model.LoginResponse{
		Token:        token,
		RefreshToken: newRefreshToken,
		User:         user.ToUserResponse(),
	}, nil
}

func LogoutService(db *sql.DB, token string) error {
	err := repository.DeleteRefreshToken(db, token)
	if err != nil {
		return errors.New("gagal logout")
	}

	return nil
}

func GetProfileService(db *sql.DB, userID string) (*model.UserResponse, error) {
	user, err := repository.FindUserByID(db, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user tidak ditemukan")
		}
		return nil, errors.New("terjadi kesalahan")
	}

	response := user.ToUserResponse()
	return &response, nil
}

func CleanupExpiredTokensService(db *sql.DB) error {
	return repository.CleanupExpiredTokens(db)
}