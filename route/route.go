package route

import (
	"github.com/gofiber/fiber/v2"
	"latihan-sintaks/app/service"
)

// Setup memetakan alamat URL ke fungsi service yang sesuai.
func Setup(app *fiber.App, studentService *service.StudentService) {
	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{"status": "ok"})
	})

	// Rute untuk mahasiswa
	students := api.Group("/students")
	students.Post("/", studentService.Create)
	students.Get("/", studentService.List)
	students.Get("/:id", studentService.Get)
	students.Put("/:id", studentService.Replace)
	students.Patch("/:id", studentService.Patch)
	students.Delete("/:id", studentService.Delete)
}