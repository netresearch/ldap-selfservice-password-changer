package rpc

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	ldap "github.com/netresearch/simple-ldap-go"
)

type Func = func(params []string) ([]string, error)

type Handler struct {
	ldap *ldap.LDAP
}

func New(ldapServer string, isActiveDirectory bool, baseDN, readonlyUser, readonlyPassword string) (*Handler, error) {
	ldap, err := ldap.New(ldapServer, baseDN, readonlyUser, readonlyPassword, isActiveDirectory)
	if err != nil {
		return nil, err
	}

	return &Handler{ldap}, nil
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

func (c *Handler) changePassword(params []string) ([]string, error) {
	if len(params) != 3 {
		return nil, ErrInvalidArgumentCount
	}

	sAMAccountName := params[0]
	oldPassword := params[1]
	newPassword := params[2]

	if oldPassword == newPassword {
		return nil, fmt.Errorf("the old password can't be same as the new one")
	}

	if err := c.ldap.ChangePasswordForSAMAccountName(sAMAccountName, oldPassword, newPassword); err != nil {
		return nil, err
	}

	return []string{"password changed successfully"}, nil
}