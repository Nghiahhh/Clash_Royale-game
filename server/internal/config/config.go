package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type ConfigStruct struct {
	MySQLHost     string
	MySQLPort     int
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string

	MongoURI string
	MongoDB  string

	WSHost string
	WSPort int

	JWTSecret string
}

// Global config biến public
var Config *ConfigStruct

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found or could not be loaded")
	}

	toInt := func(envVar string, defaultVal int) int {
		valStr := os.Getenv(envVar)
		if valStr == "" {
			return defaultVal
		}
		val, err := strconv.Atoi(valStr)
		if err != nil {
			log.Printf("Invalid value for %s: %v\n", envVar, err)
			return defaultVal
		}
		return val
	}

	Config = &ConfigStruct{
		MySQLHost:     os.Getenv("MYSQL_HOST"),
		MySQLPort:     toInt("MYSQL_PORT", 3306),
		MySQLUser:     os.Getenv("MYSQL_USER"),
		MySQLPassword: os.Getenv("MYSQL_PASSWORD"),
		MySQLDatabase: os.Getenv("MYSQL_DATABASE"),

		MongoURI: os.Getenv("MONGO_URI"),
		MongoDB:  os.Getenv("MONGO_DB"),

		WSHost:    os.Getenv("WS_HOST"),
		WSPort:    toInt("WS_PORT", 8080),
		JWTSecret: os.Getenv("JWT_SECRET"),
	}
}
