package model

import "time"

type LdapModel struct {
	ID       uint64  `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Server   string  `gorm:"column:server" json:"server"`
	Tls      *uint64 `gorm:"column:tls;default:0" json:"tls"`
	Md5      *uint64 `gorm:"column:md5;default:0" json:"md5"`
	BindDn   string  `gorm:"column:bind_dn" json:"bind_dn"`
	Password string  `gorm:"column:password" json:"password"`

	BaseDn string `gorm:"column:base_dn" json:"base_dn"`
	Filter string `gorm:"column:filter" json:"filter"`

	AdminBaseDn string `gorm:"column:admin_base_dn" json:"admin_base_dn"`
	AdminFilter string `gorm:"column:admin_filter" json:"admin_filter"`

	EmailAttr    string `gorm:"column:email_attr" json:"email_attr"`
	UserNameAttr string `gorm:"column:user_name_attr" json:"user_name_attr"`

	Enable *uint64 `gorm:"column:enable" json:"enable"`

	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`

	// sync gen use to mark user as unavailable
	SyncGen uint64 `gorm:"column:ldap_gen" json:"ldap_gen"`
	// sync protection ts use to lock the ldap sync cron job
	SyncProtectionTs int64 `gorm:"column:sync_protection_ts" json:"sync_protection_ts"`
	// if err occur when last sync, record it
	LastSyncErrMsg string `gorm:"column:last_sync_err_msg" json:"last_sync_err_msg"`

	Entries int   `gorm:"column:entries" json:"entries"`
	Inserts int   `gorm:"column:inserts" json:"inserts"`
	Updates int   `gorm:"column:updates" json:"updates"`
	Deletes int   `gorm:"column:deletes" json:"deletes"`
	Fails   int   `gorm:"column:fails" json:"fails"`
	Costs   int64 `gorm:"column:costs" json:"costs"`
}

// TableName
func (u *LdapModel) TableName() string {
	return "ldap"
}
