package repository

import (
	"database/sql"
	"go-fiber/app/model"
)


func GetAllStudents(db *sql.DB) ([]model.StudentListResponse, error) {
	rows, err := db.Query(`
		SELECT s.id, u.full_name, s.student_id, s.study_program, s.year_of_entry,
		       a.full_name AS advisor_name
		FROM students s
		JOIN users u ON s.id = u.id
		LEFT JOIN users a ON s.advisor_id = a.id
		ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []model.StudentListResponse

	for rows.Next() {
		var s model.StudentListResponse
		var advisor *string

		if err := rows.Scan(
			&s.ID,
			&s.FullName,
			&s.StudentID,
			&s.StudyProgram,
			&s.YearOfEntry,
			&advisor,
		); err != nil {
			return nil, err
		}

		s.AdvisorName = advisor

		students = append(students, s)
	}

	return students, nil
}

// func GetStudentDetail(db *sql.DB, id string) (*model.StudentDetailResponse, error) {
// 	var s model.StudentDetailResponse

// 	err := db.QueryRow(`
// 		SELECT s.id, u.full_name, s.student_id, s.study_program,
// 		       s.advisor_id,
// 		       (SELECT full_name FROM users WHERE id = s.advisor_id) AS advisor_name
// 		FROM students s
// 		JOIN users u ON s.id = u.id
// 		WHERE s.id = $1`,
// 		id).Scan(&s.ID, &s.FullName, &s.StudentID, &s.StudyProgram, &s.AdvisorID, &s.AdvisorName)

// 	if err != nil {
// 		return nil, err
// 	}

// 	return &s, nil
// }

func GetStudentByID(db *sql.DB, id string) (*model.StudentDetailResponse, error) {
	var s model.StudentDetailResponse

	err := db.QueryRow(`
		SELECT s.id, u.full_name, s.student_id, s.study_program, 
		       a.full_name AS advisor_name
		FROM students s
		JOIN users u ON s.id = u.id
		LEFT JOIN users a ON s.advisor_id = a.id
		WHERE s.id = $1
	`, id).Scan(
		&s.ID, &s.FullName, &s.StudentID, &s.StudyProgram, &s.AdvisorName,
	)

	if err != nil {
		return nil, err
	}

	return &s, nil
}

func GetLecturerByID(db *sql.DB, id string) (*model.LecturerDetailResponse, error) {
	var l model.LecturerDetailResponse

	err := db.QueryRow(`
		SELECT l.id, u.full_name, l.lecturer_id, l.department
		FROM lecturers l
		JOIN users u ON l.id = u.id
		WHERE l.id = $1
	`, id).Scan(
		&l.ID, &l.FullName, &l.LecturerID, &l.Department,
	)

	if err != nil {
		return nil, err
	}

	return &l, nil
}

func UpdateStudentAdvisor(db *sql.DB, studentID, advisorID string) error {
	_, err := db.Exec(`
		UPDATE students SET advisor_id = $1 WHERE id = $2`,
		advisorID, studentID)
	return err
}