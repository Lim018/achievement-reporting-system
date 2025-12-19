package main

import (
	"flag"
	"log"
	"os"

	"go-fiber/config"
	"go-fiber/database"
	"go-fiber/routes"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"

	_ "go-fiber/docs"
)

// @title Achievement Reporting System API
// @version 1.0
// @description Dokumentasi API untuk Sistem Pelaporan Prestasi Mahasiswa
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@university.ac.id

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3000
// @BasePath /
// @schemes http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Masukkan token dengan format: Bearer <token_jwt_anda>
func main() {
	migrate := flag.Bool("migrate", false, "Run database migrations")
	seed := flag.Bool("seed", false, "Run database seeders")
	reset := flag.Bool("reset", false, "Drop all tables, migrate, and seed (CAUTION: deletes all data)")
	flag.Parse()

	config.LoadEnv()

	db := database.ConnectDB()
	defer db.Close()

	mongoDB, err := database.ConnectMongo()
	if err != nil {
		log.Fatal("Failed to connect MongoDB:", err)
	}
	log.Println("MongoDB Connected")

	if *reset {
		log.Println("⚠️  RESETTING DATABASE - This will delete all data!")
		
		if err := database.DropTables(db); err != nil {
			log.Fatal("Failed to drop tables:", err)
		}
		
		if err := database.RunMigrations(db); err != nil {
			log.Fatal("Failed to run migrations:", err)
		}
		
		if err := database.RunSeeders(db); err != nil {
			log.Fatal("Failed to run seeders:", err)
		}
		
		log.Println("✅ Database reset completed successfully!")
		return
	}

	if *migrate {
		log.Println("Running migrations...")
		if err := database.RunMigrations(db); err != nil {
			log.Fatal("Failed to run migrations:", err)
		}
		log.Println("✅ Migrations completed successfully!")
		
		if !*seed {
			return
		}
	}

	if *seed {
		log.Println("Running seeders...")
		if err := database.RunSeeders(db); err != nil {
			log.Fatal("Failed to run seeders:", err)
		}
		log.Println("✅ Seeders completed successfully!")
		return
	}

	app := config.NewApp(db)
	app.Use(cors.New())

	app.Get("/swagger/*", swagger.HandlerDefault)

	routes.RegisterRoutes(app, db, mongoDB)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}

	log.Println("Server running on port", port)
	log.Fatal(app.Listen(":" + port))
}