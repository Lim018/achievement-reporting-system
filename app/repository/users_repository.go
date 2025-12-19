package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go-fiber/app/model"
)

type UsersRepo struct {
	DB *sql.DB
}

func NewUsersRepo(db *sql.DB) *UsersRepo {
	return &UsersRepo{DB: db}
}

func (r *UsersRepo) GetAllUsers(ctx context.Context) ([]model.UserListResponse, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT u.id, u.username, u.full_name, r.name AS role
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.is_active = true
		ORDER BY u.created_at DESC
	`)
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

func (r *UsersRepo) GetUserDetail(ctx context.Context, id string) (*model.UserDetailResponse, error) {
	var user model.UserDetailResponse

	err := r.DB.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.email, u.full_name, r.name AS role
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`, id).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &user.Role)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UsersRepo) GetUserWithRoleInfo(ctx context.Context, id string) (*model.UserDetailWithRoleInfo, error) {
	var user model.UserDetailWithRoleInfo

	err := r.DB.QueryRowContext(ctx, `
		SELECT 
			u.id, 
			u.username, 
			u.email, 
			u.full_name, 
			r.name AS role,
			u.is_active
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`, id).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &user.Role, &user.IsActive)

	if err != nil {
		return nil, err
	}

	if user.Role == "Mahasiswa" {
		var studentInfo model.StudentInfo
		err := r.DB.QueryRowContext(ctx, `
			SELECT id, student_id, study_program, year_of_entry, advisor_id
			FROM students
			WHERE user_id = $1
		`, id).Scan(
			&studentInfo.ID,
			&studentInfo.StudentID,
			&studentInfo.StudyProgram,
			&studentInfo.YearOfEntry,
			&studentInfo.AdvisorID,
		)
		if err == nil {
			user.StudentInfo = &studentInfo
		}
	}

	if user.Role == "Dosen Wali" {
		var lecturerInfo model.LecturerInfo
		err := r.DB.QueryRowContext(ctx, `
			SELECT id, lecturer_id, department
			FROM lecturers
			WHERE user_id = $1
		`, id).Scan(
			&lecturerInfo.ID,
			&lecturerInfo.LecturerID,
			&lecturerInfo.Department,
		)
		if err == nil {
			user.LecturerInfo = &lecturerInfo
		}
	}

	return &user, nil
}

func (r *UsersRepo) CreateUserTx(ctx context.Context, req model.CreateUserRequest, hashedPass string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var userID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (username, email, password_hash, full_name, role_id, is_active)
		SELECT $1, $2, $3, $4, r.id, true
		FROM roles r
		WHERE r.name = $5
		RETURNING id
	`,
		req.Username,
		req.Email,
		hashedPass,
		req.FullName,
		req.RoleName,
	).Scan(&userID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if req.RoleName == "Mahasiswa" && req.StudentID != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO students (user_id, student_id, study_program, year_of_entry, advisor_id)
			VALUES ($1, $2, $3, $4, $5)
		`,
			userID,
			req.StudentID,
			req.StudyProgram,
			req.Year,
			req.AdvisorID,
		)
		if err != nil {
			return fmt.Errorf("failed to create student record: %w", err)
		}
	}

	if req.RoleName == "Dosen Wali" && req.LecturerID != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO lecturers (user_id, lecturer_id, department)
			VALUES ($1, $2, $3)
		`,
			userID,
			req.LecturerID,
			req.Department,
		)
		if err != nil {
			return fmt.Errorf("failed to create lecturer record: %w", err)
		}
	}

	return tx.Commit()
}

func (r *UsersRepo) UpdateUserTx(ctx context.Context, id string, req model.UpdateUserRequest) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET email = $1, full_name = $2, updated_at = NOW()
		WHERE id = $3
	`, req.Email, req.FullName, id)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if req.StudentID != nil || req.StudyProgram != nil || req.YearOfEntry != nil || req.AdvisorID != nil {
		var studentExists bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM students WHERE user_id = $1)
		`, id).Scan(&studentExists)

		if err != nil {
			return fmt.Errorf("failed to check student existence: %w", err)
		}

		if studentExists {
			query := "UPDATE students SET updated_at = NOW()"
			args := []interface{}{}
			argCount := 1

			if req.StudentID != nil {
				query += fmt.Sprintf(", student_id = $%d", argCount)
				args = append(args, *req.StudentID)
				argCount++
			}
			if req.StudyProgram != nil {
				query += fmt.Sprintf(", study_program = $%d", argCount)
				args = append(args, *req.StudyProgram)
				argCount++
			}
			if req.YearOfEntry != nil {
				query += fmt.Sprintf(", year_of_entry = $%d", argCount)
				args = append(args, *req.YearOfEntry)
				argCount++
			}
			if req.AdvisorID != nil {
				query += fmt.Sprintf(", advisor_id = $%d", argCount)
				args = append(args, *req.AdvisorID)
				argCount++
			}

			query += fmt.Sprintf(" WHERE user_id = $%d", argCount)
			args = append(args, id)

			_, err = tx.ExecContext(ctx, query, args...)
			if err != nil {
				return fmt.Errorf("failed to update student: %w", err)
			}
		}
	}

	if req.LecturerID != nil || req.Department != nil {
		var lecturerExists bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM lecturers WHERE user_id = $1)
		`, id).Scan(&lecturerExists)

		if err != nil {
			return fmt.Errorf("failed to check lecturer existence: %w", err)
		}

		if lecturerExists {
			query := "UPDATE lecturers SET updated_at = NOW()"
			args := []interface{}{}
			argCount := 1

			if req.LecturerID != nil {
				query += fmt.Sprintf(", lecturer_id = $%d", argCount)
				args = append(args, *req.LecturerID)
				argCount++
			}
			if req.Department != nil {
				query += fmt.Sprintf(", department = $%d", argCount)
				args = append(args, *req.Department)
				argCount++
			}

			query += fmt.Sprintf(" WHERE user_id = $%d", argCount)
			args = append(args, id)

			_, err = tx.ExecContext(ctx, query, args...)
			if err != nil {
				return fmt.Errorf("failed to update lecturer: %w", err)
			}
		}
	}

	return tx.Commit()
}

func (r *UsersRepo) UpdateUserRole(ctx context.Context, id string, roleName string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE users
		SET role_id = (SELECT id FROM roles WHERE name = $1),
		    updated_at = NOW()
		WHERE id = $2
	`, roleName, id)
	return err
}

func (r *UsersRepo) DeleteUser(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE users 
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *UsersRepo) HardDeleteUser(ctx context.Context, id string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `DELETE FROM students WHERE user_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM lecturers WHERE user_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// func helper
func (r *UsersRepo) CheckAdvisorExists(ctx context.Context, advisorID string) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM lecturers WHERE id = $1)
	`, advisorID).Scan(&exists)
	return exists, err
}

func (r *UsersRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, username, email, full_name, role_id, is_active
		FROM users
		WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &user.RoleID, &user.IsActive)
	
	return &user, err
}

func GetAllUsers(db *sql.DB) ([]model.UserListResponse, error) {
	repo := NewUsersRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.GetAllUsers(ctx)
}

func GetUserDetail(db *sql.DB, id string) (*model.UserDetailResponse, error) {
	repo := NewUsersRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.GetUserDetail(ctx, id)
}

func CreateUserTx(db *sql.DB, req model.CreateUserRequest, hashedPass string) error {
	repo := NewUsersRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return repo.CreateUserTx(ctx, req, hashedPass)
}

func UpdateUser(db *sql.DB, id string, req model.UpdateUserRequest) error {
	repo := NewUsersRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return repo.UpdateUserTx(ctx, id, req)
}

func UpdateUserRole(db *sql.DB, id string, roleName string) error {
	repo := NewUsersRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.UpdateUserRole(ctx, id, roleName)
}

func DeleteUser(db *sql.DB, id string) error {
	repo := NewUsersRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.DeleteUser(ctx, id)
}