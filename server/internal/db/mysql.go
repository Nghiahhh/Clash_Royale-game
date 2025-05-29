package db

import (
	"database/sql"
	"fmt"
	"log"
	"server/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func InitMySQL() {
	cfg := config.Config
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.MySQLUser,
		cfg.MySQLPassword,
		cfg.MySQLHost,
		cfg.MySQLPort,
		cfg.MySQLDatabase,
	)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("MySQL connection error: %v", err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatalf("MySQL ping error: %v", err)
	}

	log.Println("✅ Connected to MySQL")
}
