package routes

import (
	c "pond/controllers"

	"github.com/gofiber/fiber/v2"
)

func Routesja(app *fiber.App) {
	app.Post("/SMTP", c.HandleAPI)
	// app.Post("/send_smtp_report", c.SendSMTPReport)

}
