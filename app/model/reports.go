package model

import "time"

type SystemStatisticsResponse struct {
	Totals            TotalsData                `json:"totals"`
	AchievementStatus map[string]int            `json:"achievement_status"`
	AchievementByType map[string]int            `json:"achievement_by_type"`
	TopStudents       []TopStudentData          `json:"top_students"`
	MonthlyGrowth     []MonthlyGrowthData       `json:"monthly_growth"`
}

type TotalsData struct {
	Students     int `json:"students"`
	Lecturers    int `json:"lecturers"`
	Achievements int `json:"achievements"`
}

type TopStudentData struct {
	StudentID  string `json:"student_id"`
	FullName   string `json:"full_name"`
	TotalPoints int   `json:"total_points"`
}

type MonthlyGrowthData struct {
	Month             string `json:"month"`
	TotalAchievements int    `json:"total_achievements"`
}

type StudentReportResponse struct {
	Student           StudentInfo               `json:"student"`
	Summary           StudentSummary            `json:"summary"`
	AchievementStatus map[string]int            `json:"achievement_status"`
	AchievementByType map[string]int            `json:"achievement_by_type"`
	MonthlyGrowth     []MonthlyGrowthData       `json:"monthly_growth"`
	Achievements      []StudentAchievementInfo  `json:"achievements"`
}

type StudentInfo struct {
	ID           string `json:"id"`
	FullName     string `json:"full_name"`
	StudentID    string `json:"student_id"`
	StudyProgram string `json:"study_program"`
	AdvisorName  string `json:"advisor_name"`
}

type StudentSummary struct {
	TotalAchievements int `json:"total_achievements"`
	TotalPoints       int `json:"total_points"`
}

type StudentAchievementInfo struct {
	ReferenceID string     `json:"reference_id"`
	MongoID     string     `json:"mongo_id"`
	Title       string     `json:"title"`
	Type        string     `json:"type"`
	Points      int        `json:"points"`
	Status      string     `json:"status"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
}