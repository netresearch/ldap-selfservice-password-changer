package rpc

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
	ldap "github.com/netresearch/simple-ldap-go"
)

type Func = func(params []string) ([]string, error)

type Handler struct {
	ldap *ldap.LDAP
	opts *options.Opts
}

func New(opts *options.Opts) (*Handler, error) {
	ldap, err := ldap.New(opts.LdapServer, opts.BaseDN, opts.ReadonlyUser, opts.ReadonlyPassword, opts.IsActiveDirectory)
	if err != nil {
		return nil, err
	}

	return &Handler{ldap, opts}, nil
}

func (h *Handler) Handle(c *fiber.Ctx) error {
	var body JSONRPC
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	wrapRPC := func(c *fiber.Ctx, fn Func) error {
		data, err := fn(body.Params)
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

	switch body.Method {
	case "change-password":
		return wrapRPC(c, h.changePassword)

	default:
		return c.Status(http.StatusBadRequest).JSON(JSONRPCResponse{
			Success: false,
			Data:    []string{"method not found"},
		})
	}
}
