package model

import "time"

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	RoleID       string    `json:"role_id"`
	Role         *Role     `json:"role,omitempty"`
	Permissions  []string  `json:"permissions,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateUserRequest struct {
    Username     string  `json:"username"`
    Email        string  `json:"email"`
    Password     string  `json:"password"`
    FullName     string  `json:"full_name"`
    RoleName     string  `json:"role_name"`

    StudentID    *string `json:"student_id,omitempty"`
    StudyProgram *string `json:"study_program,omitempty"`
    Year         *int    `json:"year,omitempty"`
    AdvisorID *string `json:"advisor_id,omitempty"`

    LecturerID   *string `json:"lecturer_id,omitempty"`
    Department   *string `json:"department,omitempty"`
}

type UpdateUserRequest struct {
    Email     string `json:"email,omitempty"`
    FullName  string `json:"full_name,omitempty"`
}

type AssignRoleRequest struct {
    RoleName string `json:"role_name" validate:"required"`
}

type UserDetailResponse struct {
    ID        string `json:"id"`
    Username  string `json:"username"`
    Email     string `json:"email"`
    FullName  string `json:"full_name"`
    Role      string `json:"role"`
}

type UserListResponse struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    FullName string `json:"full_name"`
    Role     string `json:"role"`
}

func (u *User) ToUserResponse() UserResponse {
	roleName := ""
	if u.Role != nil {
		roleName = u.Role.Name
	}
	
	return UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		FullName:    u.FullName,
		Role:        roleName,
		Permissions: u.Permissions,
	}
}