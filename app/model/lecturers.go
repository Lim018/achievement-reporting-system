package model

import "time"

type Lecturer struct {
    ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
    LecturerID string    `json:"lecturer_id"`
    Department string    `json:"department"`
    CreatedAt  time.Time `json:"created_at"`
}

type LecturerListResponse struct {
    ID         string `json:"id"`
	UserID     string `json:"user_id"`
    FullName   string `json:"full_name"`
    LecturerID string `json:"lecturer_id"`
    Department string `json:"department"`
}

type LecturerDetailResponse struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	FullName   string `json:"full_name"`
	LecturerID string `json:"lecturer_id"`
	Department string `json:"department"`
}

type LecturerAdviseeResponse struct {
    ID        string `json:"id"`
    UserID    string `json:"user_id"`
    FullName  string `json:"full_name"`
    StudentID string `json:"student_id"`
}