package ldap

import (
	"crypto/tls"
	//"crypto/tls"
	"github.com/go-ldap/ldap/v3"
	"github.com/pkg/errors"
)

// scope choices
const (
	ScopeBaseObject   = 0
	ScopeSingleLevel  = 1
	ScopeWholeSubtree = 2
)

// ScopeMap contains human readable descriptions of scope choices
var ScopeMap = map[int]string{
	ScopeBaseObject:   "Base Object",
	ScopeSingleLevel:  "Single Level",
	ScopeWholeSubtree: "Whole Subtree",
}

// derefAliases
const (
	NeverDerefAliases   = 0
	DerefInSearching    = 1
	DerefFindingBaseObj = 2
	DerefAlways         = 3
)

// DerefMap contains human readable descriptions of derefAliases choices
var DerefMap = map[int]string{
	NeverDerefAliases:   "NeverDerefAliases",
	DerefInSearching:    "DerefInSearching",
	DerefFindingBaseObj: "DerefFindingBaseObj",
	DerefAlways:         "DerefAlways",
}

func DoBindForLDAP(ldapServer string, tls, md5 bool, username, password string) error {
	return ConnectAndThen(
		ldapServer, tls, md5, username, password, func(conn *ldap.Conn) error {
			return nil
		},
	)
}

func ConnectAndThen(ldapServer string, enableTls, md5 bool, username, password string, fun func(conn *ldap.Conn) error) error {
	conn, err := ldap.DialURL(ldapServer)
	if err != nil {
		return errors.Wrap(err, "Error when connect to ldap server ")
	}
	defer conn.Close()

	if enableTls {
		err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return errors.Wrap(err, "Error when tls hand shacks to ldap server ")
		}
	}

	if conn.Bind(username, password) != nil {
		return errors.Wrap(err, "Failed to bind ")
	}

	return fun(conn)
}
