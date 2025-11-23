package database

import (
	"database/sql"
	"go-fiber/utils"
	"log"
)

// RunSeeders - Execute all database seeders
func RunSeeders(db *sql.DB) error {
	log.Println("Running seeders...")

	// Seed Roles
	if err := seedRoles(db); err != nil {
		return err
	}

	// Seed Permissions
	if err := seedPermissions(db); err != nil {
		return err
	}

	// Seed Role Permissions
	if err := seedRolePermissions(db); err != nil {
		return err
	}

	// Seed Users
	if err := seedUsers(db); err != nil {
		return err
	}

	// Seed Students & Lecturers
	if err := seedStudentsAndLecturers(db); err != nil {
		return err
	}

	log.Println("All seeders completed successfully! ✅")
	return nil
}

// seedRoles - Insert initial roles
func seedRoles(db *sql.DB) error {
	log.Println("Seeding roles...")

	roles := []struct {
		name        string
		description string
	}{
		{"Admin", "Pengelola sistem"},
		{"Mahasiswa", "Pelapor prestasi"},
		{"Dosen Wali", "Verifikator prestasi"},
	}

	for _, role := range roles {
		_, err := db.Exec(`
			INSERT INTO roles (name, description) 
			VALUES ($1, $2) 
			ON CONFLICT (name) DO NOTHING
		`, role.name, role.description)

		if err != nil {
			log.Printf("Failed to seed role %s: %v", role.name, err)
			return err
		}
	}

	log.Println("Roles seeded ✅")
	return nil
}

// seedPermissions - Insert initial permissions
func seedPermissions(db *sql.DB) error {
	log.Println("Seeding permissions...")

	permissions := []struct {
		name        string
		resource    string
		action      string
		description string
	}{
		// User permissions
		{"user:read", "user", "read", "Membaca data user"},
		{"user:create", "user", "create", "Membuat user baru"},
		{"user:update", "user", "update", "Mengupdate data user"},
		{"user:delete", "user", "delete", "Menghapus user"},

		// Role permissions
		{"role:read", "role", "read", "Membaca data role"},
		{"role:manage", "role", "manage", "Mengelola role dan permission"},

		// Prestasi permissions
		{"prestasi:read", "prestasi", "read", "Membaca data prestasi"},
		{"prestasi:create", "prestasi", "create", "Membuat prestasi baru"},
		{"prestasi:update", "prestasi", "update", "Mengupdate prestasi sendiri"},
		{"prestasi:delete", "prestasi", "delete", "Menghapus prestasi"},
		{"prestasi:verify", "prestasi", "verify", "Memverifikasi prestasi mahasiswa bimbingannya"},
		{"prestasi:manage", "prestasi", "manage", "Mengelola semua prestasi"},

		// Dashboard permissions
		{"dashboard:view", "dashboard", "view", "Melihat dashboard"},
		{"dashboard:admin", "dashboard", "admin", "Melihat dashboard admin"},
	}

	for _, perm := range permissions {
		_, err := db.Exec(`
			INSERT INTO permissions (name, resource, action, description) 
			VALUES ($1, $2, $3, $4) 
			ON CONFLICT (name) DO NOTHING
		`, perm.name, perm.resource, perm.action, perm.description)

		if err != nil {
			log.Printf("Failed to seed permission %s: %v", perm.name, err)
			return err
		}
	}

	log.Println("Permissions seeded ✅")
	return nil
}

// seedRolePermissions - Assign permissions to roles
func seedRolePermissions(db *sql.DB) error {
	log.Println("Seeding role permissions...")

	// Admin - Full access ke semua fitur
	adminPerms := []string{
		"user:read", "user:create", "user:update", "user:delete",
		"role:read", "role:manage",
		"prestasi:read", "prestasi:create", "prestasi:update", "prestasi:delete", 
		"prestasi:verify", "prestasi:manage",
		"dashboard:view", "dashboard:admin",
	}

	// Mahasiswa - Create, read, update prestasi sendiri
	mahasiswaPerms := []string{
		"prestasi:read", "prestasi:create", "prestasi:update",
		"dashboard:view",
	}

	// Dosen Wali - Read, verify prestasi mahasiswa bimbingannya
	dosenWaliPerms := []string{
		"prestasi:read", "prestasi:verify",
		"dashboard:view",
	}

	rolePermissions := map[string][]string{
		"Admin":       adminPerms,
		"Mahasiswa":   mahasiswaPerms,
		"Dosen Wali":  dosenWaliPerms,
	}

	for roleName, perms := range rolePermissions {
		for _, permName := range perms {
			_, err := db.Exec(`
				INSERT INTO role_permissions (role_id, permission_id)
				SELECT r.id, p.id
				FROM roles r, permissions p
				WHERE r.name = $1 AND p.name = $2
				ON CONFLICT DO NOTHING
			`, roleName, permName)

			if err != nil {
				log.Printf("Failed to assign permission %s to role %s: %v", permName, roleName, err)
				return err
			}
		}
	}

	log.Println("Role permissions seeded ✅")
	return nil
}

// seedUsers - Insert initial users
func seedUsers(db *sql.DB) error {
	log.Println("Seeding users...")

	// Default password: "password123"
	defaultPassword, err := utils.HashPassword("password123")
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return err
	}

	users := []struct {
		username  string
		email     string
		password  string
		fullName  string
		role      string
		isActive  bool
	}{
		{"admin", "admin@example.com", defaultPassword, "Administrator", "Admin", true},
		{"mahasiswa1", "mahasiswa1@example.com", defaultPassword, "Budi Santoso", "Mahasiswa", true},
		{"mahasiswa2", "mahasiswa2@example.com", defaultPassword, "Siti Aminah", "Mahasiswa", true},
		{"dosenwali1", "dosenwali1@example.com", defaultPassword, "Dr. Ahmad Rahman", "Dosen Wali", true},
		{"dosenwali2", "dosenwali2@example.com", defaultPassword, "Dr. Dewi Sartika", "Dosen Wali", true},
	}

	for _, user := range users {
		_, err := db.Exec(`
			INSERT INTO users (username, email, password_hash, full_name, role_id, is_active)
			SELECT $1, $2, $3, $4, r.id, $5
			FROM roles r
			WHERE r.name = $6
			ON CONFLICT (username) DO NOTHING
		`, user.username, user.email, user.password, user.fullName, user.isActive, user.role)

		if err != nil {
			log.Printf("Failed to seed user %s: %v", user.username, err)
			return err
		}
	}

	log.Println("Users seeded ✅")
	log.Println("Default credentials:")
	log.Println("  - Username: admin, Password: password123")
	log.Println("  - Username: mahasiswa1, Password: password123")
	log.Println("  - Username: dosenwali1, Password: password123")
	return nil
}

func seedStudentsAndLecturers(db *sql.DB) error {
    log.Println("Seeding students & lecturers...")

    // Example seeding
    _, err := db.Exec(`
        INSERT INTO students (id, student_id, study_program, year_of_entry)
        SELECT u.id, 'NIM001', 'Teknik Informatika', 2022
        FROM users u
        JOIN roles r ON u.role_id = r.id
        WHERE u.username = 'mahasiswa1'
        ON CONFLICT (id) DO NOTHING;
    `)
    if err != nil { return err }

    _, err = db.Exec(`
        INSERT INTO lecturers (id, lecturer_id, department)
        SELECT u.id, 'NIP001', 'Teknik Informatika'
        FROM users u
        JOIN roles r ON u.role_id = r.id
        WHERE u.username = 'dosenwali1'
        ON CONFLICT (id) DO NOTHING;
    `)
    if err != nil { return err }

    log.Println("Students & Lecturers seeded ✓")
    return nil
}