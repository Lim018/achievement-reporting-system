package database

import (
    "database/sql"
    "log"
    "os"
    _ "github.com/lib/pq"
)

func ConnectDB() *sql.DB {
    dsn := os.Getenv("DB_DSN")
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        log.Fatal(err)
    }

    if err = db.Ping(); err != nil {
        log.Fatal("Database tidak connect:", err)
    }

    log.Println("DB Connected")
    return db
}
