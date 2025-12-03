package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

func RegisterRoutes(app *fiber.App, db *sql.DB, mongoDB *mongo.Database) {
    UserRoutes(app, db)
    StudentRoutes(app, db)
    LecturerRoutes(app, db)

}
