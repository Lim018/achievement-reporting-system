package repository

import (
	"database/sql"
	"go-fiber/app/model"
)

func GetAllLecturers(db *sql.DB) ([]model.LecturerListResponse, error) {
	rows, err := db.Query(`
		SELECT 
			l.id,
			l.user_id,
			u.full_name,
			l.lecturer_id,
			l.department
		FROM lecturers l
		JOIN users u ON l.user_id = u.id
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

func GetLecturerAdvisees(db *sql.DB, lecturerID string) ([]model.LecturerAdviseeResponse, error) {
	rows, err := db.Query(`
		SELECT 
			s.id,
			s.user_id,
			u.full_name,
			s.student_id
		FROM students s
		JOIN users u ON u.id = s.user_id
		WHERE s.advisor_id = $1
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