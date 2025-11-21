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

	refreshToken, _, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, errors.New("gagal generate refresh token")
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

    user, err := repository.FindUserByID(db, claims.UserID)
    if err != nil {
        return nil, errors.New("user tidak ditemukan")
    }

    token, err := utils.GenerateToken(*user)
    if err != nil {
        return nil, errors.New("gagal generate token")
    }

    refreshToken, _, err := utils.GenerateRefreshToken(user.ID)
    if err != nil {
        return nil, errors.New("gagal generate refresh token")
    }

    return &model.LoginResponse{
        Token:        token,
        RefreshToken: refreshToken,
        User:         user.ToUserResponse(),
    }, nil
}

func LogoutService(db *sql.DB, token string) error {
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