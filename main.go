package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/core"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

var (
	fLdapServer        = flag.String("ldap-server", "ldaps://localhost:636", "LDAP server URI, has to begin with `ldap://` or `ldaps://`. If this is an ActiveDirectory server, this *has* to be `ldaps://`.")
	fIsActiveDirectory = flag.Bool("active-directory", false, "Mark the LDAP server as ActiveDirectory.")
	fBaseDN            = flag.String("base-dn", "", "Base DN of your LDAP directory.")
	fReadonlyUser      = flag.String("readonly-user", "", "User that can read all users in your LDAP directory.")
	fReadonlyPassword  = flag.String("readonly-password", "", "Password for the readonly user")

	methodChangePassword string = "change-password"
)

type JSONRPC struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type JSONRPCResponse struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
}

func wrapRPC(c *fiber.Ctx, params []string, fn core.RPCFunc) error {
	data, err := fn(params)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(JSONRPCResponse{
			Success: false,
			Data:    []string{err.Error()},
		})
	}

	return c.JSON(JSONRPCResponse{
		Success: true,
		Data:    data,
	})
}

func main() {
	flag.Parse()

	coreApi, err := core.New(*fLdapServer, *fIsActiveDirectory, *fBaseDN, *fReadonlyUser, *fReadonlyPassword)
	if err != nil {
		log.Fatalf("An error occurred during initialization: %v", err)
	}

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
		case methodChangePassword:
			return wrapRPC(c, body.Params, coreApi.ChangePassword)

		default:
			return c.Status(http.StatusBadRequest).JSON(JSONRPCResponse{
				Success: false,
				Data:    []string{"method not found"},
			})
		}
	})

	app.Listen(":3000")
}
