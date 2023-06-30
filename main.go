package main

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/rpc"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

func main() {
	opts := options.Parse()

	rpcHandler, err := rpc.New(opts)
	if err != nil {
		log.Fatalf("An error occurred during initialization: %v", err)
	}

	views := html.NewFileSystem(http.FS(templates.Templates), ".html")
	views.AddFunc("InputOpts", templates.MakeInputOpts)

	app := fiber.New(fiber.Config{
		Views: views,
	})

	app.Use("/static", filesystem.New(filesystem.Config{
		Root: http.FS(static.Static),
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"opts": opts,
		})
	})

	app.Post("/api/rpc", rpcHandler.Handle)

	app.Listen(":3000")
}
