package service

import (
	"database/sql"
	"go-fiber/app/model"
	"go-fiber/app/repository"
	"go-fiber/utils"

	"github.com/gofiber/fiber/v2"
)

func GetAllUsersService(c *fiber.Ctx, db *sql.DB) error {
	users, err := repository.GetAllUsers(db)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data users",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   users,
	})
}

func GetUserDetailService(c *fiber.Ctx, db *sql.DB) error {
	id := c.Params("id")
	user, err := repository.FindUserByID(db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(model.APIResponse{
				Status: "error",
				Error:  "User tidak ditemukan",
			})
		}
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Terjadi kesalahan saat mengambil user",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   user.ToUserResponse(),
	})
}

func CreateUserService(c *fiber.Ctx, db *sql.DB) error {
	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "Request body tidak valid",
		})
	}

	if req.Username == "" || req.Password == "" || req.Email == "" || req.FullName == "" || req.RoleName == "" {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "Username, password, email, full_name, role_name wajib diisi",
		})
	}

	roleLower := req.RoleName
	if roleLower == "Mahasiswa" || roleLower == "mahasiswa" || roleLower == "Student" || roleLower == "student" {
		if req.StudentID == nil || *req.StudentID == "" {
			return c.Status(400).JSON(model.APIResponse{
				Status: "error",
				Error:  "student_id wajib diisi saat membuat user dengan role Mahasiswa",
			})
		}
	}
	if roleLower == "Dosen Wali" || roleLower == "dosen wali" || roleLower == "Lecturer" || roleLower == "lecturer" {
		if req.LecturerID == nil || *req.LecturerID == "" {
			return c.Status(400).JSON(model.APIResponse{
				Status: "error",
				Error:  "lecturer_id wajib diisi saat membuat user dengan role Dosen Wali",
			})
		}
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal memproses password",
		})
	}

	userID, err := repository.CreateUser(db, req, hashed)
	if err != nil {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  err.Error(),
		})
	}

	if roleLower == "Mahasiswa" || roleLower == "mahasiswa" || roleLower == "Student" || roleLower == "student" {
		if err := repository.CreateStudent(db, userID, *req.StudentID, req.RoleName); err != nil {
			_ = repository.DeleteUser(db, userID)
			return c.Status(500).JSON(model.APIResponse{
				Status: "error",
				Error:  "Gagal membuat profil mahasiswa: " + err.Error(),
			})
		}
	}

	if roleLower == "Dosen Wali" || roleLower == "dosen wali" || roleLower == "Lecturer" || roleLower == "lecturer" {
		if err := repository.CreateLecturer(db, userID, *req.LecturerID, req.RoleName); err != nil {
			_ = repository.DeleteUser(db, userID)
			return c.Status(500).JSON(model.APIResponse{
				Status: "error",
				Error:  "Gagal membuat profil dosen: " + err.Error(),
			})
		}
	}

	user, err := repository.FindUserByID(db, userID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "User dibuat tetapi gagal mengambil data user",
		})
	}

	return c.Status(201).JSON(model.APIResponse{
		Status:  "success",
		Message: "User berhasil dibuat",
		Data:    user.ToUserResponse(),
	})
}

func UpdateUserService(c *fiber.Ctx, db *sql.DB) error {
	id := c.Params("id")

	var req model.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "Request body tidak valid",
		})
	}

	if err := repository.UpdateUser(db, id, req); err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal memperbarui user: " + err.Error(),
		})
	}

	user, err := repository.FindUserByID(db, id)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "User diperbarui tetapi gagal mengambil data",
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "User berhasil diperbarui",
		Data:    user.ToUserResponse(),
	})
}

func DeleteUserService(c *fiber.Ctx, db *sql.DB) error {
	id := c.Params("id")

	if err := repository.DeleteUser(db, id); err != nil {
		return c.Status(404).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal menghapus user: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "User berhasil dihapus",
	})
}

func AssignRoleService(c *fiber.Ctx, db *sql.DB) error {
	id := c.Params("id")

	var body struct {
		RoleName   string  `json:"role_name"`
		StudentID  *string `json:"student_id,omitempty"`
		LecturerID *string `json:"lecturer_id,omitempty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "Request body tidak valid",
		})
	}

	if body.RoleName == "" {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "role_name wajib diisi",
		})
	}

	roleLower := body.RoleName
	if roleLower == "Mahasiswa" || roleLower == "mahasiswa" || roleLower == "Student" || roleLower == "student" {
		if body.StudentID == nil || *body.StudentID == "" {
			return c.Status(400).JSON(model.APIResponse{
				Status: "error",
				Error:  "student_id wajib diisi saat assign role Mahasiswa",
			})
		}
	}
	if roleLower == "Dosen Wali" || roleLower == "dosen wali" || roleLower == "Lecturer" || roleLower == "lecturer" {
		if body.LecturerID == nil || *body.LecturerID == "" {
			return c.Status(400).JSON(model.APIResponse{
				Status: "error",
				Error:  "lecturer_id wajib diisi saat assign role Dosen Wali",
			})
		}
	}

	if err := repository.UpdateUserRole(db, id, body.RoleName); err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengubah role: " + err.Error(),
		})
	}

	if roleLower == "Mahasiswa" || roleLower == "mahasiswa" || roleLower == "Student" || roleLower == "student" {
		exists, err := repository.StudentExists(db, id)
		if err != nil {
			return c.Status(500).JSON(model.APIResponse{
				Status: "error",
				Error:  "Gagal cek profil mahasiswa: " + err.Error(),
			})
		}
		if !exists {
			if err := repository.CreateStudent(db, id, *body.StudentID, body.RoleName); err != nil {
				return c.Status(500).JSON(model.APIResponse{
					Status: "error",
					Error:  "Gagal membuat profil mahasiswa: " + err.Error(),
				})
			}
		}
	}

	if roleLower == "Dosen Wali" || roleLower == "dosen wali" || roleLower == "Lecturer" || roleLower == "lecturer" {
		exists, err := repository.LecturerExists(db, id)
		if err != nil {
			return c.Status(500).JSON(model.APIResponse{
				Status: "error",
				Error:  "Gagal cek profil dosen: " + err.Error(),
			})
		}
		if !exists {
			if err := repository.CreateLecturer(db, id, *body.LecturerID, body.RoleName); err != nil {
				return c.Status(500).JSON(model.APIResponse{
					Status: "error",
					Error:  "Gagal membuat profil dosen: " + err.Error(),
				})
			}
		}
	}

	user, err := repository.FindUserByID(db, id)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Role diupdate tetapi gagal mengambil user: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "Role berhasil diupdate",
		Data:    user.ToUserResponse(),
	})
}

// students
func GetAllStudentsService(c *fiber.Ctx, db *sql.DB) error {
	students, err := repository.GetAllStudents(db)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data mahasiswa: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   students,
	})
}

func GetStudentDetailService(c *fiber.Ctx, db *sql.DB) error {
	id := c.Params("id")
	st, err := repository.GetStudentByID(db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(model.APIResponse{
				Status: "error",
				Error:  "Mahasiswa tidak ditemukan",
			})
		}
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil mahasiswa: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   st,
	})
}

func UpdateStudentAdvisorService(c *fiber.Ctx, db *sql.DB) error {
	studentID := c.Params("id")
	var req model.UpdateStudentAdvisorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "Request body tidak valid",
		})
	}

	if exists, err := repository.LecturerExists(db, req.AdvisorID); err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal cek dosen: " + err.Error(),
		})
	} else if !exists {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "Dosen advisor tidak ditemukan",
		})
	}

	if err := repository.UpdateStudentAdvisor(db, studentID, req.AdvisorID); err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengupdate advisor: " + err.Error(),
		})
	}

	updated, _ := repository.GetStudentByID(db, studentID)
	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "Advisor berhasil diupdate",
		Data:    updated,
	})
}

// lecturers
func GetAllLecturersService(c *fiber.Ctx, db *sql.DB) error {
	lecturers, err := repository.GetAllLecturers(db)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data dosen: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   lecturers,
	})
}

func GetLecturerAdviseesService(c *fiber.Ctx, db *sql.DB) error {
	lecturerID := c.Params("id")
	advisees, err := repository.GetAdvisees(db, lecturerID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data bimbingan: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   advisees,
	})
}