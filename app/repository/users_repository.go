package repository

import (
	"database/sql"
	"go-fiber/app/model"
)

func GetAllUsers(db *sql.DB) ([]model.UserListResponse, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, u.full_name, r.name AS role
		FROM users u
		JOIN roles r ON u.role_id = r.id
		ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.UserListResponse

	for rows.Next() {
		var u model.UserListResponse
		if err := rows.Scan(&u.ID, &u.Username, &u.FullName, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func GetUserDetail(db *sql.DB, id string) (*model.UserDetailResponse, error) {
	var user model.UserDetailResponse

	err := db.QueryRow(`
		SELECT u.id, u.username, u.email, u.full_name, r.name AS role
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1`,
		id).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &user.Role)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func CreateUserTx(db *sql.DB, req model.CreateUserRequest, hashedPass string) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }

    _, err = tx.Exec(`
        INSERT INTO users (username, email, password_hash, full_name, role_id)
        SELECT $1, $2, $3, $4, r.id
        FROM roles r
        WHERE r.name = $5
    `,
        req.Username, req.Email, hashedPass, req.FullName, req.RoleName,
    )
    if err != nil {
        tx.Rollback()
        return err
    }

    var userID string
    err = tx.QueryRow(`SELECT id FROM users WHERE username = $1`, req.Username).Scan(&userID)
    if err != nil {
        tx.Rollback()
        return err
    }

    if req.StudentID != nil {
        _, err = tx.Exec(`
            INSERT INTO students (id, student_id, study_program, year_of_entry)
            VALUES ($1, $2, $3, $4)
        `,
            userID,
            *req.StudentID,
            req.StudyProgram,
            req.Year,
        )
        if err != nil {
            tx.Rollback()
            return err
        }
    }

    if req.LecturerID != nil {
        _, err = tx.Exec(`
            INSERT INTO lecturers (id, lecturer_id, department)
            VALUES ($1, $2, $3)
        `,
            userID,
            *req.LecturerID,
            req.Department,
        )
        if err != nil {
            tx.Rollback()
            return err
        }
    }

    return tx.Commit()
}

func UpdateUser(db *sql.DB, id string, req model.UpdateUserRequest) error {
	_, err := db.Exec(`
		UPDATE users
		SET email = $1, full_name = $2, updated_at = NOW()
		WHERE id = $3`,
		req.Email, req.FullName, id)
	return err
}

func UpdateUserRole(db *sql.DB, id string, roleName string) error {
	_, err := db.Exec(`
		UPDATE users
		SET role_id = (SELECT id FROM roles WHERE name = $1),
		    updated_at = NOW()
		WHERE id = $2`,
		roleName, id)
	return err
}

func DeleteUser(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}