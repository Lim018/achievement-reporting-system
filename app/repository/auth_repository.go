package repository

import (
	"database/sql"
	"go-fiber/app/model"
	"time"
)

func FindUserByUsernameOrEmail(db *sql.DB, identifier string) (*model.User, string, error) {
	var user model.User
	var role model.Role
	var passwordHash string

	err := db.QueryRow(`
		SELECT u.id, u.username, u.email, u.password_hash, u.full_name, 
		       u.role_id, u.is_active, u.created_at, u.updated_at,
		       r.id, r.name, r.description, r.created_at
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE (u.username = $1 OR u.email = $1) AND u.is_active = true
	`, identifier).Scan(
		&user.ID, &user.Username, &user.Email, &passwordHash, &user.FullName,
		&user.RoleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&role.ID, &role.Name, &role.Description, &role.CreatedAt,
	)

	if err != nil {
		return nil, "", err
	}

	user.Role = &role

	permissions, err := GetUserPermissions(db, user.RoleID)
	if err != nil {
		return nil, "", err
	}
	user.Permissions = permissions

	return &user, passwordHash, nil
}

func GetUserPermissions(db *sql.DB, roleID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT p.name
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`, roleID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

func FindUserByID(db *sql.DB, userID string) (*model.User, error) {
	var user model.User
	var role model.Role

	err := db.QueryRow(`
		SELECT u.id, u.username, u.email, u.full_name, 
		       u.role_id, u.is_active, u.created_at, u.updated_at,
		       r.id, r.name, r.description, r.created_at
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1 AND u.is_active = true
	`, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.FullName,
		&user.RoleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&role.ID, &role.Name, &role.Description, &role.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	user.Role = &role

	permissions, err := GetUserPermissions(db, user.RoleID)
	if err != nil {
		return nil, err
	}
	user.Permissions = permissions

	return &user, nil
}

func SaveRefreshToken(db *sql.DB, userID, token string, expiresAt time.Time) error {
	_, err := db.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`, userID, token, expiresAt)

	return err
}

func FindRefreshToken(db *sql.DB, token string) (*model.RefreshToken, error) {
	var rt model.RefreshToken

	err := db.QueryRow(`
		SELECT id, user_id, token, expires_at, created_at
		FROM refresh_tokens
		WHERE token = $1 AND expires_at > NOW()
	`, token).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt, &rt.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &rt, nil
}

func DeleteRefreshToken(db *sql.DB, token string) error {
	_, err := db.Exec(`
		DELETE FROM refresh_tokens WHERE token = $1
	`, token)

	return err
}

func DeleteUserRefreshTokens(db *sql.DB, userID string) error {
	_, err := db.Exec(`
		DELETE FROM refresh_tokens WHERE user_id = $1
	`, userID)

	return err
}

func CleanupExpiredTokens(db *sql.DB) error {
	_, err := db.Exec(`
		DELETE FROM refresh_tokens WHERE expires_at < NOW()
	`)

	return err
}