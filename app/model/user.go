package model

type CreateUserRequest struct {
	Username    string `json:"username" validate:"required"`
	Email       string `json:"email" validate:"required"`
	Password    string `json:"password" validate:"required"`
	FullName    string `json:"full_name" validate:"required"`
	RoleName    string `json:"role_name" validate:"required"`
	StudentID   *string `json:"student_id,omitempty"`
	LecturerID  *string `json:"lecturer_id,omitempty"`
}

type UpdateUserRequest struct {
	Email       string `json:"email,omitempty"`
	FullName    string `json:"full_name,omitempty"`
	RoleName    string `json:"role_name,omitempty"`
}

type AssignRoleRequest struct {
	RoleName string `json:"role_name" validate:"required"`
}

type UpdateStudentAdvisorRequest struct {
	AdvisorID string `json:"advisor_id" validate:"required"`
}

type UserListResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	Role      string `json:"role"`
}