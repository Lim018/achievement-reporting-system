package test

import (
	"database/sql"
	"os"
	"regexp"
	"testing"
	"time"

	"go-fiber/app/model"
	"go-fiber/app/service"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes)
}

func TestLoginService(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret_key_123")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	fixedTime := time.Now()
	hashedPass := hashPassword("password123")

	userCols := []string{
		"id", "username", "email", "password_hash", "full_name",
		"role_id", "is_active", "created_at", "updated_at",
		"id", "name", "description", "created_at",
	}

	permCols := []string{"name"}

	t.Run("Login Success", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT u.id, u.username")).
			WithArgs("testuser").
			WillReturnRows(sqlmock.NewRows(userCols).AddRow(
				"user-uuid-1", "testuser", "test@email.com", hashedPass, "Test User",
				"role-uuid-1", true, fixedTime, fixedTime,
				"role-uuid-1", "Mahasiswa", "Desc", fixedTime,
			))

		mock.ExpectQuery(regexp.QuoteMeta("SELECT p.name FROM permissions")).
			WithArgs("role-uuid-1").
			WillReturnRows(sqlmock.NewRows(permCols).AddRow("achievement:create").AddRow("achievement:read"))

		req := model.LoginRequest{Username: "testuser", Password: "password123"}
		resp, err := service.LoginService(db, req)

		assert.NoError(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, "testuser", resp.User.Username)
			assert.NotEmpty(t, resp.Token)
			assert.NotEmpty(t, resp.RefreshToken)
		}
	})

	t.Run("User Not Found", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT u.id, u.username")).
			WithArgs("unknown").
			WillReturnError(sql.ErrNoRows)

		req := model.LoginRequest{Username: "unknown", Password: "password123"}
		resp, err := service.LoginService(db, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "username atau email tidak ditemukan", err.Error())
	})

	t.Run("Wrong Password", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT u.id, u.username")).
			WithArgs("testuser").
			WillReturnRows(sqlmock.NewRows(userCols).AddRow(
				"user-uuid-1", "testuser", "test@email.com", hashedPass, "Test User",
				"role-uuid-1", true, fixedTime, fixedTime,
				"role-uuid-1", "Mahasiswa", "Desc", fixedTime,
			))

		mock.ExpectQuery(regexp.QuoteMeta("SELECT p.name FROM permissions")).
			WithArgs("role-uuid-1").
			WillReturnRows(sqlmock.NewRows(permCols).AddRow("achievement:read"))

		req := model.LoginRequest{Username: "testuser", Password: "wrongpassword"}
		resp, err := service.LoginService(db, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "password salah", err.Error())
	})

	t.Run("Account Inactive", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT u.id, u.username")).
			WithArgs("inactive_user").
			WillReturnRows(sqlmock.NewRows(userCols).AddRow(
				"user-uuid-2", "inactive_user", "inactive@email.com", hashedPass, "Inactive User",
				"role-uuid-1", false, fixedTime, fixedTime,
				"role-uuid-1", "Mahasiswa", "Desc", fixedTime,
			))

		mock.ExpectQuery(regexp.QuoteMeta("SELECT p.name FROM permissions")).
			WithArgs("role-uuid-1").
			WillReturnRows(sqlmock.NewRows(permCols))

		req := model.LoginRequest{Username: "inactive_user", Password: "password123"}
		resp, err := service.LoginService(db, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "akun tidak aktif", err.Error())
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}