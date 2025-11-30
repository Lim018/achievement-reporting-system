package database

import (
	"database/sql"
	"log"
)

// RunMigrations - Execute all database migrations
func RunMigrations(db *sql.DB) error {
	log.Println("Running migrations...")

	migrations := []string{
		// Create roles table
		`CREATE TABLE IF NOT EXISTS roles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(50) UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create permissions table
		`CREATE TABLE IF NOT EXISTS permissions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) UNIQUE NOT NULL,
			resource VARCHAR(50) NOT NULL,
			action VARCHAR(50) NOT NULL,
			description TEXT
		)`,

		// Create role_permissions junction table
		`CREATE TABLE IF NOT EXISTS role_permissions (
			role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
			PRIMARY KEY (role_id, permission_id)
		)`,

		// Create users table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			full_name VARCHAR(100) NOT NULL,
			role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create lecturers table
		`CREATE TABLE IF NOT EXISTS lecturers (
			id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			lecturer_id VARCHAR(50) UNIQUE NOT NULL,
			department VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create students table
		`CREATE TABLE IF NOT EXISTS students (
			id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			student_id VARCHAR(50) UNIQUE NOT NULL,
			study_program VARCHAR(100),
			year_of_entry INT,
			advisor_id UUID REFERENCES lecturers(id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,


		// Create index for better performance
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_students_student_id ON students(student_id);`,
		`CREATE INDEX IF NOT EXISTS idx_students_advisor_id ON students(advisor_id);`,
		`CREATE INDEX IF NOT EXISTS idx_lecturers_lecturer_id ON lecturers(lecturer_id);`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			log.Printf("Migration %d failed: %v", i+1, err)
			return err
		}
		log.Printf("Migration %d completed ✅", i+1)
	}

	log.Println("All migrations completed successfully! ✅")
	return nil
}

// DropTables - Drop all tables (use with caution!)
func DropTables(db *sql.DB) error {
	log.Println("Dropping all tables...")

	drops := []string{
		`DROP TABLE IF EXISTS users CASCADE`,
		`DROP TABLE IF EXISTS role_permissions CASCADE`,
		`DROP TABLE IF EXISTS permissions CASCADE`,
		`DROP TABLE IF EXISTS roles CASCADE`,
		`DROP TABLE IF EXISTS lecturers CASCADE`,
		`DROP TABLE IF EXISTS students CASCADE`,
	}

	for _, drop := range drops {
		if _, err := db.Exec(drop); err != nil {
			log.Printf("Drop table failed: %v", err)
			return err
		}
	}

	log.Println("All tables dropped successfully! ✅")
	return nil
}