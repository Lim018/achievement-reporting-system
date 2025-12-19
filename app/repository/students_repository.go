package repository

import (
	"context"
	"database/sql"
	"time"

	"go-fiber/app/model"
)

type StudentsRepo struct {
	DB *sql.DB
}

func NewStudentsRepo(db *sql.DB) *StudentsRepo {
	return &StudentsRepo{DB: db}
}

func (r *StudentsRepo) GetAllStudents(ctx context.Context) ([]model.StudentListResponse, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT 
			s.id,
			s.user_id,
			u.full_name,
			s.student_id,
			s.study_program,
			s.year_of_entry,
			u_advisor.full_name AS advisor_name
		FROM students s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN lecturers l ON s.advisor_id = l.id
		LEFT JOIN users u_advisor ON l.user_id = u_advisor.id
		WHERE u.is_active = true
		ORDER BY s.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.StudentListResponse
	for rows.Next() {
		var s model.StudentListResponse
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.FullName,
			&s.StudentID,
			&s.StudyProgram,
			&s.YearOfEntry,
			&s.AdvisorName,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	return out, nil
}

func (r *StudentsRepo) GetStudentByID(ctx context.Context, studentID string) (*model.StudentDetailResponse, error) {
	var s model.StudentDetailResponse

	err := r.DB.QueryRowContext(ctx, `
		SELECT 
			s.id,
			s.user_id,
			u.full_name,
			s.student_id,
			s.study_program,
			s.year_of_entry,
			s.advisor_id,
			u_advisor.full_name AS advisor_name
		FROM students s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN lecturers l ON s.advisor_id = l.id
		LEFT JOIN users u_advisor ON l.user_id = u_advisor.id
		WHERE s.id = $1
	`, studentID).Scan(
		&s.ID,
		&s.UserID,
		&s.FullName,
		&s.StudentID,
		&s.StudyProgram,
		&s.YearOfEntry,
		&s.AdvisorID,
		&s.AdvisorName,
	)

	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *StudentsRepo) GetStudentByUserID(ctx context.Context, userID string) (*model.StudentDetailResponse, error) {
	var s model.StudentDetailResponse

	err := r.DB.QueryRowContext(ctx, `
		SELECT 
			s.id,
			s.user_id,
			u.full_name,
			s.student_id,
			s.study_program,
			s.year_of_entry,
			s.advisor_id,
			u_advisor.full_name AS advisor_name
		FROM students s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN lecturers l ON s.advisor_id = l.id
		LEFT JOIN users u_advisor ON l.user_id = u_advisor.id
		WHERE s.user_id = $1
	`, userID).Scan(
		&s.ID,
		&s.UserID,
		&s.FullName,
		&s.StudentID,
		&s.StudyProgram,
		&s.YearOfEntry,
		&s.AdvisorID,
		&s.AdvisorName,
	)

	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *StudentsRepo) UpdateStudentAdvisor(ctx context.Context, studentID, advisorID string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE students
		SET advisor_id = $1, updated_at = NOW()
		WHERE id = $2
	`, advisorID, studentID)
	return err
}

// func helper
func (r *StudentsRepo) GetStudentIDByUserID(ctx context.Context, userID string) (string, error) {
	var studentID string
	err := r.DB.QueryRowContext(ctx, `
		SELECT id FROM students WHERE user_id = $1
	`, userID).Scan(&studentID)
	return studentID, err
}

func GetAllStudents(db *sql.DB) ([]model.StudentListResponse, error) {
	repo := NewStudentsRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.GetAllStudents(ctx)
}

func GetStudentByID(db *sql.DB, studentID string) (*model.StudentDetailResponse, error) {
	repo := NewStudentsRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.GetStudentByID(ctx, studentID)
}

func UpdateStudentAdvisor(db *sql.DB, studentID, advisorID string) error {
	repo := NewStudentsRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.UpdateStudentAdvisor(ctx, studentID, advisorID)
}

func GetStudentIDByUserID(db *sql.DB, userID string) (string, error) {
	repo := NewStudentsRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return repo.GetStudentIDByUserID(ctx, userID)
}