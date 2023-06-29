package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/rpc"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/templates"
)

var (
	fLdapServer        = flag.String("ldap-server", "ldaps://localhost:636", "LDAP server URI, has to begin with `ldap://` or `ldaps://`. If this is an ActiveDirectory server, this *has* to be `ldaps://`.")
	fIsActiveDirectory = flag.Bool("active-directory", false, "Mark the LDAP server as ActiveDirectory.")
	fBaseDN            = flag.String("base-dn", "", "Base DN of your LDAP directory.")
	fReadonlyUser      = flag.String("readonly-user", "", "User that can read all users in your LDAP directory.")
	fReadonlyPassword  = flag.String("readonly-password", "", "Password for the readonly user")
)

func main() {
	flag.Parse()

	rpcHandler, err := rpc.New(*fLdapServer, *fIsActiveDirectory, *fBaseDN, *fReadonlyUser, *fReadonlyPassword)
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

	app.Post("/api/rpc", rpcHandler.Handle)

	app.Listen(":3000")
}
