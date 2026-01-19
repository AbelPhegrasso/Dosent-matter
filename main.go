package main

import (
	r "good/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	r.Routesja(app)
	app.Listen(":8888")
}
