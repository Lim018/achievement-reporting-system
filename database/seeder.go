package database

import (
	"database/sql"
	"go-fiber/utils"
	"log"
)

func RunSeeders(db *sql.DB) error {
	log.Println("Running seeders...")

	if err := seedRoles(db); err != nil {
		return err
	}

	if err := seedPermissions(db); err != nil {
		return err
	}

	if err := seedRolePermissions(db); err != nil {
		return err
	}

	if err := seedUsers(db); err != nil {
		return err
	}
	
	log.Println("All seeders completed successfully! ✅")
	return nil
}

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

func seedPermissions(db *sql.DB) error {
	log.Println("Seeding permissions...")

	permissions := []struct {
		name        string
		resource    string
		action      string
		description string
	}{
		{"user:manage", "user", "manage", "Mengelola data user"},
		{"achievement:create", "achievement", "create", "Membuat prestasi baru"},
		{"achievement:read", "achievement", "read", "Membaca prestasi"},
		{"achievement:update", "achievement", "update", "Mengupdate prestasi"},
		{"achievement:delete", "achievement", "delete", "Menghapus prestasi"},
		{"achievement:verify", "achievement", "verify", "Memverifikasi prestasi mahasiswa"},
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

func seedRolePermissions(db *sql.DB) error {
	log.Println("Seeding role permissions...")

	adminPerms := []string{
		"user:manage",
		"achievement:read",
		"achievement:create",
		"achievement:update",
		"achievement:delete",
		"achievement:verify",
	}

	mahasiswaPerms := []string{
		"achievement:read",
		"achievement:create",
		"achievement:update",
		"achievement:delete",
	}

	dosenWaliPerms := []string{
		"achievement:read",
		"achievement:verify",
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

func seedUsers(db *sql.DB) error {
	log.Println("Seeding users...")

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
		{"admin", "admin@gmail.com", defaultPassword, "Administrator", "Admin", true},
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
	return nil
}