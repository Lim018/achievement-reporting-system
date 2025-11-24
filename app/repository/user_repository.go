package repository

import (
	"database/sql"
	"errors"
	"go-fiber/app/model"
)

// users
func CreateUser(db *sql.DB, req model.CreateUserRequest, hashedPass string) (string, error) {
	var id string

	err := db.QueryRow(`
		INSERT INTO users (username, email, password_hash, full_name, role_id)
		VALUES ($1, $2, $3, $4,
			(SELECT id FROM roles WHERE name = $5)
		)
		RETURNING id
	`, req.Username, req.Email, hashedPass, req.FullName, req.RoleName).Scan(&id)

	if err != nil {
		return "", err
	}

	return id, nil
}

func GetAllUsers(db *sql.DB) ([]model.UserListResponse, error) {
	rows, err := db.Query(`
        SELECT u.id, u.username, u.full_name, r.name
        FROM users u
        JOIN roles r ON u.role_id = r.id
        ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []model.UserListResponse{}
	for rows.Next() {
		var user model.UserListResponse
		if err := rows.Scan(&user.ID, &user.Username, &user.FullName, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func UpdateUser(db *sql.DB, userID string, req model.UpdateUserRequest) error {
	_, err := db.Exec(`
		UPDATE users
		SET email = $1, full_name = $2
		WHERE id = $3
	`, req.Email, req.FullName, userID)

	return err
}

func DeleteUser(db *sql.DB, userID string) error {
	_, err := db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	return err
}

func UpdateUserRole(db *sql.DB, userID string, roleName string) error {
	res, err := db.Exec(`
		UPDATE users SET role_id = (SELECT id FROM roles WHERE name = $1)
		WHERE id = $2
	`, roleName, userID)

	aff, _ := res.RowsAffected()
	if aff == 0 {
		return errors.New("user tidak ditemukan")
	}

	return err
}

// students

func CreateStudent(db *sql.DB, userID, studentID, program string) error {
	_, err := db.Exec(`
		INSERT INTO students (id, student_id, study_program)
		VALUES ($1, $2, $3)
	`, userID, studentID, program)
	return err
}

func StudentExists(db *sql.DB, userID string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM students WHERE id = $1)
	`, userID).Scan(&exists)
	return exists, err
}

func GetAllStudents(db *sql.DB) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT s.id, u.full_name, s.student_id, s.study_program,
		       l.full_name AS advisor_name
		FROM students s
		JOIN users u ON s.id = u.id
		LEFT JOIN users l ON s.advisor_id = l.id
		ORDER BY u.full_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var id, fullName, studentID, program, advisorName sql.NullString

		if err := rows.Scan(&id, &fullName, &studentID, &program, &advisorName); err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"id":            id.String,
			"full_name":     fullName.String,
			"student_id":    studentID.String,
			"study_program": program.String,
			"advisor_name":  advisorName.String,
		})
	}

	return results, nil
}

func GetStudentByID(db *sql.DB, studentID string) (map[string]interface{}, error) {
	row := db.QueryRow(`
		SELECT s.id, u.full_name, s.student_id, s.study_program, s.advisor_id
		FROM students s
		JOIN users u ON s.id = u.id
		WHERE s.id = $1
	`, studentID)

	var id, fullName, stdID, program, advisorID sql.NullString

	if err := row.Scan(&id, &fullName, &stdID, &program, &advisorID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":            id.String,
		"full_name":     fullName.String,
		"student_id":    stdID.String,
		"study_program": program.String,
		"advisor_id":    advisorID.String,
	}, nil
}

func UpdateStudentAdvisor(db *sql.DB, studentID, lecturerID string) error {
	_, err := db.Exec(`
		UPDATE students SET advisor_id = $1 WHERE id = $2
	`, lecturerID, studentID)
	return err
}

// lecturers

func CreateLecturer(db *sql.DB, userID, lecturerID, department string) error {
	_, err := db.Exec(`
		INSERT INTO lecturers (id, lecturer_id, department)
		VALUES ($1, $2, $3)
	`, userID, lecturerID, department)
	return err
}

func LecturerExists(db *sql.DB, userID string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM lecturers WHERE id = $1)
	`, userID).Scan(&exists)
	return exists, err
}

func GetAllLecturers(db *sql.DB) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT l.id, u.full_name, l.lecturer_id, l.department
		FROM lecturers l
		JOIN users u ON l.id = u.id
		ORDER BY u.full_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var id, name, lid, dept sql.NullString
		if err := rows.Scan(&id, &name, &lid, &dept); err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"id":          id.String,
			"full_name":   name.String,
			"lecturer_id": lid.String,
			"department":  dept.String,
		})
	}

	return results, nil
}

func GetAdvisees(db *sql.DB, lecturerID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT s.id, u.full_name, s.student_id
		FROM students s
		JOIN users u ON s.id = u.id
		WHERE s.advisor_id = $1
	`, lecturerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var id, fullName, studentID sql.NullString
		if err := rows.Scan(&id, &fullName, &studentID); err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"id":         id.String,
			"full_name":  fullName.String,
			"student_id": studentID.String,
		})
	}

	return results, nil
}