package ldap

import (
	"github.com/jinzhu/gorm"
	"nocalhost/internal/nocalhost-api/model"
)

type LdapRepo struct {
	db *gorm.DB
}

func NewLdapRepo(db *gorm.DB) *LdapRepo {
	return &LdapRepo{
		db: db,
	}
}

// Close close db
func (repo *LdapRepo) Close() {
	repo.db.Close()
}

func (repo *LdapRepo) SelectFirstOne() (*model.LdapModel, error) {
	data := new(model.LdapModel)

	err := repo.db.First(data).Error
	return data, err
}

func (repo *LdapRepo) MarkErrorOccur(id, syncGen uint64, errMsg string) bool {
	return repo.db.Exec(
		"UPDATE ldap SET last_sync_err_msg = ? WHERE id = ? and ldap_gen = ?",
		errMsg, id, syncGen,
	).RowsAffected > 0
}

func (repo *LdapRepo) TryLock(id uint64, newProtectedTs, nowProtectedTs int64) bool {
	return repo.db.Exec(
		"UPDATE ldap SET sync_protection_ts = ? WHERE id = ? and sync_protection_ts = ?",
		newProtectedTs, id, nowProtectedTs,
	).RowsAffected > 0
}

func (repo *LdapRepo) TryUnLock(id uint64, nowProtectedTs int64) {
	repo.db.Exec(
		"UPDATE ldap SET sync_protection_ts = ? WHERE id = ? and sync_protection_ts = ?",
		0, id, nowProtectedTs,
	)
}

func (repo *LdapRepo) CreateOrUpdate(update map[string]interface{}) error {
	return repo.db.Transaction(
		func(tx *gorm.DB) error {
			data := new(model.LdapModel)
			err := tx.First(data).Error

			// create if not exist
			if err != nil {
				if gorm.IsRecordNotFoundError(err) {

					// create one
					if err := tx.Create(data).Error; err != nil {
						return err
					}
				} else {
					return err
				}
			}

			update["id"] = data.ID
			return tx.Model(data).Updates(update, false).Error
		},
	)
}

func (repo *LdapRepo) UpdateGen(id uint64, gen uint64, entries int, inserts int, updates int, deletes, fails int,
	costs int64) error {
	return repo.db.Exec(
		"UPDATE ldap SET ldap_gen = ?, entries = ?, inserts = ?, "+
			"updates = ?, deletes = ?, fails = ?, costs = ?, last_sync_err_msg = null"+
			" WHERE id = ?", gen, entries, inserts, updates, deletes, fails, costs, id,
	).Error
}
