package ldap

import (
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/ldap"
)

type LdapService interface {
	Configuration(
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
		Enable *uint64) error
	Get() (*model.LdapModel, error)
	TryGetLock(id uint64, newProtectedTs, nowProtectedTs int64) bool
	TryUnLock(id uint64, nowProtectedTs int64)
	MarkErrorOccur(id, syncGen uint64, errMsg string) bool
	UpdateGen(id, syncGen uint64, entries, inserts, updates, deletes, fails int, costs int64, ) error
}

type ldapService struct {
	ldapRepo *ldap.LdapRepo
}

func NewLdapService() LdapService {
	db := model.GetDB()
	return &ldapService{ldapRepo: ldap.NewLdapRepo(db)}
}

func (srv *ldapService) Configuration(
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

func (srv *ldapService) Get() (*model.LdapModel, error) {
	return srv.ldapRepo.SelectFirstOne()
}

func (srv *ldapService) TryGetLock(id uint64, newProtectedTs, nowProtectedTs int64) bool {
	return srv.ldapRepo.TryLock(id, newProtectedTs, nowProtectedTs)
}

func (srv *ldapService) TryUnLock(id uint64, nowProtectedTs int64) {
	srv.ldapRepo.TryUnLock(id, nowProtectedTs)
}

func (srv *ldapService) MarkErrorOccur(id, syncGen uint64, errMsg string) bool {
	return srv.ldapRepo.MarkErrorOccur(id, syncGen, errMsg)
}

func (srv *ldapService) UpdateGen(id, syncGen uint64, entries, inserts, updates, deletes, fails int, costs int64) error {
	return srv.ldapRepo.UpdateGen(id, syncGen, entries, inserts, updates, deletes, fails, costs)
}

func AppendToMapIfNotEmpty(m map[string]interface{}, k string, v interface{}) {
	if v != "" {
		m[k] = v
	}
}
