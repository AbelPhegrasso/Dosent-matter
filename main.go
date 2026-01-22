package main

import (
	"log"
	r "pond/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	app := fiber.New()
	r.Routesja(app)
	app.Listen(":8888")
}
