package ldap

import (
	"github.com/gin-gonic/gin"
	ldap2 "github.com/go-ldap/ldap/v3"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"math/rand"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/ldap"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nocalhost-api/pkg/utils"
	"strings"

	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

func GetConfiguration(c *gin.Context) {
	if configuration, err := service.Svc.LdapSvc.Get(); err != nil {
		if gorm.IsRecordNotFoundError(err) {

			api.SendResponse(c, nil, &model.LdapModel{})
			return
		} else {

			log.Errorf("Fail to getting ldap config, Error %s", err)
			api.SendResponse(c, errno.ErrFailToGetLDAPSettings, err.Error())
			return
		}
	} else {
		if _const.Uint64PointerToBool(configuration.Enable) {
			api.SendResponse(c, nil, configuration)
		} else {
			api.SendResponse(c, nil, model.LdapModel{
				Enable: _const.BoolToUint64Pointer(false),
			})
		}
	}
}

func Configuration(c *gin.Context) {
	var params Configurations

	if err := c.ShouldBindJSON(&params); err != nil {
		api.SendResponse(c, errno.ErrBind, err.Error())
		return
	}

	if err := service.Svc.LdapSvc.Configuration(
		params.Server,
		params.Tls,
		params.Md5,
		params.BindDn,
		params.Password,
		params.BaseDn,
		params.Filter,
		params.AdminBaseDn,
		params.AdminFilter,
		params.EmailAttr,
		params.UserNameAttr,
		_const.BoolToUint64Pointer(true),
	); err != nil {
		log.Errorf("Fail to saving ldap config, Error %s", err)
		api.SendResponse(c, errno.ErrFailToSaveLDAPSettings, err.Error())
		return
	}

	api.SendResponse(c, nil, nil)
}

func DeleteConfiguration(c *gin.Context) {
	if err := service.Svc.LdapSvc.Configuration(
		"",
		nil,
		nil,
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		_const.BoolToUint64Pointer(false),
	); err != nil {
		log.Errorf("Fail to disable ldap config, Error %s", err)
		api.SendResponse(c, errno.ErrFailToSaveLDAPSettings, err.Error())
		return
	}

	api.SendResponse(c, nil, nil)
}

func TestBind(c *gin.Context) {
	var params Bind

	if err := c.ShouldBindJSON(&params); err != nil {
		api.SendResponse(c, errno.ErrBind, err.Error())
		return
	}

	if err := ldap.DoBindForLDAP(
		params.Server,
		_const.Uint64PointerToBool(params.Tls),
		_const.Uint64PointerToBool(params.Md5),
		params.BindDn,
		params.Password,
	); err != nil {
		log.Errorf("Ldap binding fail, Error %s", err)
		api.SendResponse(c, errno.ErrFailLDAPBind, err.Error())
		return
	}

	api.SendResponse(c, nil, nil)
}

func TestSearch(c *gin.Context) {
	var params Configurations

	if err := c.ShouldBindJSON(&params); err != nil {
		api.SendResponse(c, errno.ErrBind, err.Error())
		return
	}

	var entry *ldap2.Entry

	err := ldap.ConnectAndThen(
		params.Server,
		_const.Uint64PointerToBool(params.Tls),
		_const.Uint64PointerToBool(params.Md5),
		params.BindDn,
		params.Password,
		func(conn *ldap2.Conn) error {

			// >> preparation for search
			if params.Filter == "" {
				params.Filter = "(objectClass=*)"
			}

			searchRequest := ldap2.NewSearchRequest(
				params.BaseDn,
				ldap.ScopeWholeSubtree, ldap.DerefAlways, 0, 0, false,
				params.Filter,
				[]string{
					"dn", params.EmailAttr, params.UserNameAttr,
				},
				nil,
				//[]ldap2.Control{ldap2.NewControlPaging(5)},
			)

			sr, err := conn.Search(searchRequest)
			if err != nil {
				return errors.Wrap(err, "Error when sending search request to ldap server ")
			}

			if len(sr.Entries) > 0 {
				entry = sr.Entries[rand.Intn(len(sr.Entries))]
				return nil
			} else {
				entry = nil
				return errors.New("Nothing fetched from ldap server")
			}
		},
	)
	if err != nil {
		log.ErrorE(err, "Error while test search ")
		api.SendResponse(c, errno.ErrFailToSearchLDAP, err.Error())
		return
	}

	email := entry.GetAttributeValue(params.EmailAttr)

	userName := entry.GetAttributeValue(params.UserNameAttr)
	if userName == "" && utils.IsEmail(email) {
		userName = email[:strings.Index(email, "@")]
	}

	api.SendResponse(c, nil, &FakeUser{
		Email:    email,
		UserName: userName,
		BindDn:   entry.DN,
	})
}

func Trigger(c *gin.Context) {
	service.CronJobTrigger()
	api.SendResponse(c, nil, nil)
}

type FakeUser struct {
	Email    string `json:"email"`
	UserName string `json:"user_name"`
	BindDn   string `json:"bind_dn"`
}

type Bind struct {
	Server   string  `json:"server" binding:"required"`
	Tls      *uint64 `json:"tls"`
	Md5      *uint64 `json:"md5"`
	BindDn   string  `json:"bind_dn" binding:"required"`
	Password string  `json:"password" binding:"required"`
}

type Configurations struct {
	Server       string  `json:"server" binding:"required"`
	Tls          *uint64 `json:"tls"`
	Md5          *uint64 `json:"md5"`
	BindDn       string  `json:"bind_dn" binding:"required"`
	Password     string  `json:"password" binding:"required"`
	BaseDn       string  `json:"base_dn" binding:"required"`
	Filter       string  `json:"filter"`
	AdminBaseDn  string  `json:"admin_base_dn"`
	AdminFilter  string  `json:"admin_filter"`
	EmailAttr    string  `json:"email_attr" binding:"required"`
	UserNameAttr string  `json:"user_name_attr"`
}
