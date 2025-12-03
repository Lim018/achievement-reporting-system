package database

import (
    "context"
    "database/sql"
    "log"
    "os"
    "time"

    _ "github.com/lib/pq"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var MongoDB *mongo.Database

func ConnectDB() *sql.DB {
    dsn := os.Getenv("DB_DSN")
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        log.Fatal(err)
    }

    if err = db.Ping(); err != nil {
        log.Fatal("Database tidak connect:", err)
    }

    log.Println("PostgreSQL Connected")
    return db
}

func ConnectMongo() (*mongo.Database, error) {
    uri := os.Getenv("MONGO_URI")
    dbName := os.Getenv("MONGO_DB")

    if uri == "" || dbName == "" {
        log.Fatal("MONGO_URI atau MONGO_DB belum diset di .env")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }

    if err := client.Ping(ctx, nil); err != nil {
        return nil, err
    }

    MongoClient = client
    MongoDB = client.Database(dbName)

    log.Println("MongoDB Connected")
    return MongoDB, nil
}