package core

import (
	ldap "github.com/netresearch/simple-ldap-go"
)

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
