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

	config.LoadEnv()

	db := database.ConnectDB()
	defer db.Close()

	app := config.NewApp(db)

	routes.RegisterRoutes(app, db)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}