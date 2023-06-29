package core

import (
	"errors"
	"fmt"

	ldap "github.com/netresearch/simple-ldap-go"
)

var ErrInvalidArgumentCount = errors.New("invalid argument count")

type RPCFunc = func(params []string) ([]string, error)

type Core struct {
	ldap *ldap.LDAP
}

func New(ldapServer string, isActiveDirectory bool, baseDN, readonlyUser, readonlyPassword string) (*Core, error) {
	ldap, err := ldap.New(ldapServer, baseDN, readonlyUser, readonlyPassword, isActiveDirectory)
	if err != nil {
		return nil, err
	}

	return &Core{ldap}, nil
}

func (c *Core) ChangePassword(params []string) ([]string, error) {
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
