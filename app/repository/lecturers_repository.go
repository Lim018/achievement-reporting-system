package repository

import (
	"context"
	"database/sql"
	"time"

	"go-fiber/app/model"
)

type LecturersRepo struct {
	DB *sql.DB
}

func NewLecturersRepo(db *sql.DB) *LecturersRepo {
	return &LecturersRepo{DB: db}
}

func (r *LecturersRepo) GetAllLecturers(ctx context.Context) ([]model.LecturerListResponse, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT 
			l.id,
			l.user_id,
			u.full_name,
			l.lecturer_id,
			l.department
		FROM lecturers l
		JOIN users u ON l.user_id = u.id
		WHERE u.is_active = true
		ORDER BY u.full_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.LecturerListResponse
	for rows.Next() {
		var l model.LecturerListResponse
		if err := rows.Scan(
			&l.ID,
			&l.UserID,
			&l.FullName,
			&l.LecturerID,
			&l.Department,
		); err != nil {
			return nil, err
		}
		list = append(list, l)
	}

	return list, nil
}

func (r *LecturersRepo) GetLecturerByID(ctx context.Context, id string) (*model.LecturerDetailResponse, error) {
	var l model.LecturerDetailResponse

	err := r.DB.QueryRowContext(ctx, `
		SELECT 
			l.id, 
			l.user_id,
			u.full_name, 
			l.lecturer_id, 
			l.department
		FROM lecturers l
		JOIN users u ON l.user_id = u.id
		WHERE l.id = $1
	`, id).Scan(
		&l.ID, 
		&l.UserID,
		&l.FullName, 
		&l.LecturerID, 
		&l.Department,
	)

	if err != nil {
		return nil, err
	}

	return &l, nil
}

func (r *LecturersRepo) GetLecturerAdvisees(ctx context.Context, lecturerID string) ([]model.LecturerAdviseeResponse, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT 
			s.id,
			s.user_id,
			u.full_name,
			s.student_id
		FROM students s
		JOIN users u ON u.id = s.user_id
		WHERE s.advisor_id = $1 AND u.is_active = true
		ORDER BY u.full_name
	`, lecturerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.LecturerAdviseeResponse
	for rows.Next() {
		var a model.LecturerAdviseeResponse
		if err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.FullName,
			&a.StudentID,
		); err != nil {
			return nil, err
		}
		list = append(list, a)
	}

	return list, nil
}

func (r *LecturersRepo) GetLecturerAdviseesByUserID(ctx context.Context, userID string) ([]model.LecturerAdviseeResponse, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT 
			s.id,
			s.user_id,
			u.full_name,
			s.student_id
		FROM students s
		JOIN users u ON u.id = s.user_id
		JOIN lecturers l ON s.advisor_id = l.id
		WHERE l.user_id = $1 AND u.is_active = true
		ORDER BY u.full_name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.LecturerAdviseeResponse
	for rows.Next() {
		var a model.LecturerAdviseeResponse
		if err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.FullName,
			&a.StudentID,
		); err != nil {
			return nil, err
		}
		list = append(list, a)
	}

	return list, nil
}

func GetAllLecturers(db *sql.DB) ([]model.LecturerListResponse, error) {
	repo := NewLecturersRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.GetAllLecturers(ctx)
}

func GetLecturerByID(db *sql.DB, id string) (*model.LecturerDetailResponse, error) {
	repo := NewLecturersRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.GetLecturerByID(ctx, id)
}

func GetLecturerAdvisees(db *sql.DB, lecturerID string) ([]model.LecturerAdviseeResponse, error) {
	repo := NewLecturersRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.GetLecturerAdvisees(ctx, lecturerID)
}