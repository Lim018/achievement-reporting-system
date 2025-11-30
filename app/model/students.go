package model

import "time"

type Student struct {
    ID           string    `json:"id"`
    StudentID    string    `json:"student_id"`
    StudyProgram string    `json:"study_program"`
    AdvisorID    *string   `json:"advisor_id,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type StudentListResponse struct {
    ID           string  `json:"id"`
    FullName     string  `json:"full_name"`
    StudentID    string  `json:"student_id"`
    StudyProgram string  `json:"study_program"`
    AdvisorID    *string `json:"advisor_id,omitempty"`
    AdvisorName  *string `json:"advisor_name,omitempty"`
}

type StudentDetailResponse struct {
    ID           string  `json:"id"`
    FullName     string  `json:"full_name"`
    StudentID    string  `json:"student_id"`
    StudyProgram string  `json:"study_program"`
    AdvisorID    *string `json:"advisor_id,omitempty"`
    AdvisorName  *string `json:"advisor_name,omitempty"`
}

type UpdateStudentAdvisorRequest struct {
    AdvisorID string `json:"advisor_id" validate:"required"`
}