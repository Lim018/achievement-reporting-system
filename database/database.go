package database

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var MongoClient *mongo.Client
var MongoDB *mongo.Database

func ConnectDB() *sql.DB {
	dsn := os.Getenv("DB_DSN")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	maxOpenConns := getEnvAsInt("PG_MAX_OPEN_CONNS", 25)
	maxIdleConns := getEnvAsInt("PG_MAX_IDLE_CONNS", 10)
	connMaxLifetime := getEnvAsDuration("PG_CONN_MAX_LIFETIME", 5*time.Minute)
	connMaxIdleTime := getEnvAsDuration("PG_CONN_MAX_IDLE_TIME", 5*time.Minute)

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		log.Fatal("Database tidak connect:", err)
	}

	// log.Println("   - PostgreSQL Connected")
	// log.Printf("   - MaxOpenConns: %d", maxOpenConns)
	// log.Printf("   - MaxIdleConns: %d", maxIdleConns)
	// log.Printf("   - ConnMaxLifetime: %v", connMaxLifetime)
	// log.Printf("   - ConnMaxIdleTime: %v", connMaxIdleTime)

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

	maxPoolSize := uint64(getEnvAsInt("MONGO_MAX_POOL_SIZE", 100))
	minPoolSize := uint64(getEnvAsInt("MONGO_MIN_POOL_SIZE", 10))
	maxConnIdleTime := getEnvAsDuration("MONGO_MAX_CONN_IDLE_TIME", 5*time.Minute)
	connectTimeout := getEnvAsDuration("MONGO_CONNECT_TIMEOUT", 10*time.Second)
	serverSelTimeout := getEnvAsDuration("MONGO_SERVER_SEL_TIMEOUT", 5*time.Second)

	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(maxPoolSize).
		SetMinPoolSize(minPoolSize).
		SetMaxConnIdleTime(maxConnIdleTime).
		SetConnectTimeout(connectTimeout).
		SetServerSelectionTimeout(serverSelTimeout).
		SetRetryWrites(true).
		SetRetryReads(true).
		SetCompressors([]string{"snappy", "zlib"})

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	MongoClient = client
	MongoDB = client.Database(dbName)

	// log.Println("   - MongoDB Connected")
	// log.Printf("   - Database: %s", dbName)
	// log.Printf("   - MaxPoolSize: %d", maxPoolSize)
	// log.Printf("   - MinPoolSize: %d", minPoolSize)
	// log.Printf("   - MaxConnIdleTime: %v", maxConnIdleTime)

	return MongoDB, nil
}

func CloseDatabases(pg *sql.DB) {
	log.Println("Shutting down database connections")

	if pg != nil {
		if err := pg.Close(); err != nil {
			log.Printf("Error closing PostgreSQL: %v", err)
		} else {
			log.Println("PostgreSQL connection closed")
		}
	}

	if MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := MongoClient.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		} else {
			log.Println("MongoDB connection closed")
		}
	}
}

func GetDBStats(pg *sql.DB) map[string]interface{} {
	stats := pg.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	return defaultVal
}