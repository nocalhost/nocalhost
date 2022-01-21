package ldap

import (
	"crypto/tls"
	"fmt"
	"sync"

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

var lock = sync.Mutex{}
var connectionMap = make(map[string]*ldap.Conn)

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

func lockFor(do func() (*ldap.Conn, error)) (*ldap.Conn, error) {
	lock.Lock()
	defer lock.Unlock()
	return do()
}

func genKey(ldapServer string, enableTls, md5 bool, username, password string) string {
	return fmt.Sprintf("%s-%v-%v-%s-%s", ldapServer, enableTls, md5, username, password)
}

func GetOrNewFromPool(ldapServer string, enableTls, md5 bool, username, password string) (*ldap.Conn, error) {
	key := genKey(ldapServer, enableTls, md5, username, password)

	return lockFor(
		func() (*ldap.Conn, error) {
			conn := connectionMap[key]
			if conn != nil {
				if !conn.IsClosing() {
					return conn, nil
				}

				delete(connectionMap, key)
			}

			conn, err := ldap.DialURL(ldapServer)
			needDestroy := true
			defer func() {
				if needDestroy {
					conn.Close()
				}
			}()

			if err != nil {
				return nil, errors.Wrap(err, "Error when connect to ldap server ")
			}

			if enableTls {
				err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
				if err != nil {
					return nil, errors.Wrap(err, "Error when tls hand shacks to ldap server ")
				}
			}

			if md5 {
				if err := conn.MD5Bind(ldapServer, username, password); err != nil {
					return nil, errors.Wrap(err, "Md5 bind failed ")
				}
			} else {
				if conn.Bind(username, password) != nil {
					return nil, errors.Wrap(err, "Bind failed ")
				}
			}

			needDestroy = false
			connectionMap[key] = conn
			return conn, nil
		},
	)
}

func ConnectAndThen(ldapServer string, enableTls, md5 bool, username, password string, fun func(conn *ldap.Conn) error) error {
	if conn, err := GetOrNewFromPool(ldapServer, enableTls, md5, username, password); err != nil {
		return err
	} else {
		return fun(conn)
	}
}
