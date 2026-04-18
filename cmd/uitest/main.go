//go:build uitest

// Ad-hoc dev server for Playwright / browser UI testing. Renders the real
// templates and serves the real static assets, without LDAP/SMTP.
// Run with: `go run -tags=uitest ./cmd/uitest`. Not built in normal builds.
package main

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	webstatic "github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

func main() {
	opts := &options.Opts{
		Port:                        "3333",
		MinLength:                   10,
		MinNumbers:                  2,
		MinSymbols:                  1,
		MinUppercase:                1,
		MinLowercase:                1,
		PasswordCanIncludeUsername:  false,
		PasswordResetEnabled:        true,
		ResetTokenExpiryMinutes:     15,
		ResetRateLimitRequests:      3,
		ResetRateLimitWindowMinutes: 60,
	}

	index, err := templates.RenderIndex(opts)
	if err != nil {
		log.Fatalf("render index: %v", err)
	}
	forgot, err := templates.RenderForgotPassword()
	if err != nil {
		log.Fatalf("render forgot: %v", err)
	}
	reset, err := templates.RenderResetPassword(opts)
	if err != nil {
		log.Fatalf("render reset: %v", err)
	}

	app := fiber.New()
	app.Use("/static", static.New("", static.Config{FS: webstatic.Static}))
	app.Get("/", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(index)
	})
	app.Get("/forgot-password", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(forgot)
	})
	app.Get("/reset-password", func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMETextHTMLCharsetUTF8)
		return c.Send(reset)
	})
	app.Post("/api/rpc", func(c fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(fiber.Map{"success": true, "data": []string{}})
	})
	log.Println("listening on :3333")
	if err := app.Listen(":3333"); err != nil {
		log.Fatal(err)
	}
}
