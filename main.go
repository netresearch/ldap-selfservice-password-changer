package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
	"github.com/netresearch/ldap-selfservice-password-changer/web"
	"github.com/netresearch/ldap-selfservice-password-changer/web/static"
)

func main() {
	views := html.NewFileSystem(http.FS(web.Templates), ".html")

	app := fiber.New(fiber.Config{
		Views: views,
	})

	app.Use("/static", filesystem.New(filesystem.Config{
		Root: http.FS(static.Static),
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{})
	})

	app.Listen(":3000")
}
