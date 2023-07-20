package main

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
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
		AppName:   "netresearch/ldap-selfservice-password-changer",
		BodyLimit: 4 * 1024,
		Views:     views,
	})

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	app.Use("/static", filesystem.New(filesystem.Config{
		Root:   http.FS(static.Static),
		MaxAge: 24 * 60 * 60,
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"opts": opts,
		})
	})

	app.Post("/api/rpc", rpcHandler.Handle)

	if err := app.Listen(":3000"); err != nil {
		log.Printf("err: could not start web server: %s", err)
	}
}
