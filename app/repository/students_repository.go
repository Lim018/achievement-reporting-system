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

func GetStudentByID(db *sql.DB, id string) (*model.StudentDetailResponse, error) {
	var s model.StudentDetailResponse

	err := db.QueryRow(`
		SELECT s.id, u.full_name, s.student_id, s.study_program, s.year_of_entry,
		       a.full_name AS advisor_name
		FROM students s
		JOIN users u ON s.id = u.id
		LEFT JOIN users a ON s.advisor_id = a.id
		WHERE s.id = $1
	`, id).Scan(
		&s.ID, &s.FullName, &s.StudentID, &s.StudyProgram, &s.YearOfEntry, &s.AdvisorName,
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

func (r *AchievementRefRepo) ListByStudentID(studentID string) ([]model.AchievementDetailResponse, error) {
	rows, err := r.PG.Query(`
        SELECT id, mongo_achievement_id, status,
               submitted_at, verified_at, verified_by, rejection_note,
               created_at, updated_at
        FROM achievement_references
        WHERE student_id = $1 AND status != 'deleted'
        ORDER BY created_at DESC
    `, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.AchievementDetailResponse{}
	for rows.Next() {
		var item model.AchievementDetailResponse
		var submittedAt, verifiedAt sql.NullTime
		var verifiedBy, rejectionNote sql.NullString
		var mongoHex string

		err := rows.Scan(
			&item.ReferenceID,
			&mongoHex,
			&item.ReferenceStatus,
			&submittedAt,
			&verifiedAt,
			&verifiedBy,
			&rejectionNote,
			&item.CreatedAtRef,
			&item.UpdatedAtRef,
		)
		if err != nil {
			return nil, err
		}

		item.MongoID = mongoHex

		if submittedAt.Valid {
			item.SubmittedAt = &submittedAt.Time
		}
		if verifiedAt.Valid {
			item.VerifiedAt = &verifiedAt.Time
		}
		if verifiedBy.Valid {
			s := verifiedBy.String
			item.VerifiedBy = &s
		}
		if rejectionNote.Valid {
			s := rejectionNote.String
			item.RejectionNote = &s
		}

		out = append(out, item)
	}

	return out, nil
}

func UpdateStudentAdvisor(db *sql.DB, studentID, advisorID string) error {
	_, err := db.Exec(`
		UPDATE students SET advisor_id = $1 WHERE id = $2`,
		advisorID, studentID)
	return err
}