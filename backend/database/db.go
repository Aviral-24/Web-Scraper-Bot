package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var DB *sql.DB

func ConnectDB() {
	
	
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Warning: No .env file found (Using server environment variables)")
	}

	// 2. .env se credentials nikalna
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	// 3. MySQL ki connection string dynamically banana
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPass, dbHost, dbPort, dbName)

	// 4. Database se connect karna
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("❌ Database connection string error: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Println("⚠️ MySQL not connected! Is your database running?")
	} else {
		log.Println("🛢️ [Database] MySQL Connected Successfully!")
	}
}
