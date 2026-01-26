package main

import (
	"fmt"
	"log"
	"os"
	"pond/database"
	r "pond/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func initDatabase() {

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		os.Getenv("DATABASE_USER"),
		"",
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("DATABASE_NAME"),
	)
	var err error
	database.DBConn, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Database connected!")
}
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	initDatabase()
	app := fiber.New()
	r.Routesja(app)
	app.Listen(":8888")
}
