package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

type JSONRPC struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type JSONRPCResponse struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
}

func main() {
	views := html.NewFileSystem(http.FS(templates.Templates), ".html")

	app := fiber.New(fiber.Config{
		Views: views,
	})

	app.Use("/static", filesystem.New(filesystem.Config{
		Root: http.FS(static.Static),
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{})
	})

	app.Post("/api/rpc", func(c *fiber.Ctx) error {
		var body JSONRPC
		if err := c.BodyParser(&body); err != nil {
			return err
		}

		switch body.Method {
		default:
			return c.Status(http.StatusNotFound).JSON(JSONRPCResponse{
				Success: false,
				Data:    []string{"method not found"},
			})
		}
	})

	app.Listen(":3000")
}
