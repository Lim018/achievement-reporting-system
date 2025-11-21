package main

import (
	"flag"
	"log"
	"os"
	"go-fiber/config"
	"go-fiber/database"
	"go-fiber/routes"
)

func main() {
	migrate := flag.Bool("migrate", false, "Run database migrations")
	seed := flag.Bool("seed", false, "Run database seeders")
	reset := flag.Bool("reset", false, "Drop all tables, migrate, and seed (CAUTION: deletes all data)")
	flag.Parse()

	config.LoadEnv()

	db := database.ConnectDB()
	defer db.Close()

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

	routes.RegisterRoutes(app, db)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}