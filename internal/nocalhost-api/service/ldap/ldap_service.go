package ldap

import (
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/ldap"
)

type Ldap struct {
	ldapRepo *ldap.LdapRepo
}

func NewLdapService() *Ldap {
	db := model.GetDB()
	return &Ldap{ldapRepo: ldap.NewLdapRepo(db)}
}

func (srv *Ldap) Configuration(
	Server string,
	Tls *uint64,
	Md5 *uint64,
	BindDn string,
	Password string,
	BaseDn string,
	Filter string,
	AdminBaseDn string,
	AdminFilter string,
	EmailAttr string,
	UserNameAttr string,
	Enable *uint64) error {
	mapping := make(map[string]interface{}, 0)

	AppendToMapIfNotEmpty(mapping, "server", Server)
	AppendToMapIfNotEmpty(mapping, "tls", Tls)
	AppendToMapIfNotEmpty(mapping, "md5", Md5)
	AppendToMapIfNotEmpty(mapping, "bindDn", BindDn)
	AppendToMapIfNotEmpty(mapping, "password", Password)
	AppendToMapIfNotEmpty(mapping, "baseDn", BaseDn)
	AppendToMapIfNotEmpty(mapping, "filter", Filter)
	AppendToMapIfNotEmpty(mapping, "adminBaseDn", AdminBaseDn)
	AppendToMapIfNotEmpty(mapping, "adminFilter", AdminFilter)
	AppendToMapIfNotEmpty(mapping, "emailAttr", EmailAttr)
	AppendToMapIfNotEmpty(mapping, "userNameAttr", UserNameAttr)
	AppendToMapIfNotEmpty(mapping, "enable", Enable)

	return srv.ldapRepo.CreateOrUpdate(mapping)
}

func (srv *Ldap) Get() (*model.LdapModel, error) {
	return srv.ldapRepo.SelectFirstOne()
}

func (srv *Ldap) TryGetLock(id uint64, newProtectedTs, nowProtectedTs int64) bool {
	return srv.ldapRepo.TryLock(id, newProtectedTs, nowProtectedTs)
}

func (srv *Ldap) TryUnLock(id uint64, nowProtectedTs int64) {
	srv.ldapRepo.TryUnLock(id, nowProtectedTs)
}

func (srv *Ldap) MarkErrorOccur(id, syncGen uint64, errMsg string) bool {
	return srv.ldapRepo.MarkErrorOccur(id, syncGen, errMsg)
}

func (srv *Ldap) UpdateGen(id, syncGen uint64, entries, inserts, updates, deletes, fails int, costs int64) error {
	return srv.ldapRepo.UpdateGen(id, syncGen, entries, inserts, updates, deletes, fails, costs)
}

func AppendToMapIfNotEmpty(m map[string]interface{}, k string, v interface{}) {
	if v != "" {
		m[k] = v
	}
}
