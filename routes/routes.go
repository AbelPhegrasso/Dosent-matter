package routes

import (
	c "good/controllers"

	"github.com/gofiber/fiber/v2"
)

func Routesja(app *fiber.App) {
	app.Post("/getToken", c.GenToken)
	app.Post("/send_smtp_report", c.SendSMTPReport)

}
