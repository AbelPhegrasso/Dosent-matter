package routes

import (
	c "good/controllers"

	"github.com/gofiber/fiber/v2"
)

func Routesja(app *fiber.App) {
	app.Post("/getToken", c.HandleAPI)
	// app.Post("/send_smtp_report", c.SendSMTPReport)

}
