package service

import (
	"context"
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"k8s.io/apimachinery/pkg/util/sets"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/model"
	ldapsrv "nocalhost/internal/nocalhost-api/service/ldap"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var once = sync.Once{}
var running = atomic.Value{}
var lock = sync.Mutex{}

var RUNNING = "RUNNING"
var IDLE = "IDLE"

func StartJob() {
	go func() {
		once.Do(
			func() {
				tickC := time.Tick(time.Minute * 30)

				for {
					if err := recover(); err != nil {
						log.Error("Panic while exec cron job ")
					}
					CronJobTrigger()

					select {
					case <-tickC:
					}
				}
			},
		)
	}()
}

func CronJobTrigger() {
	go cronJobTrigger()
}

func cronJobTrigger() {
	if loaded := running.Load(); loaded != nil && loaded == RUNNING {
		return
	}

	// double check
	lock.Lock()
	if loaded := running.Load(); loaded != nil && loaded == RUNNING {
		lock.Unlock()
		return
	}

	running.Store(RUNNING)
	lock.Unlock()
	defer running.Store(IDLE)

	var err error
	var ldapModel *model.LdapModel

	log.Info("CronJob for LDAP is triggered")

	ldapModel, err = Svc.LdapSvc.Get()
	if err == nil {
		newProtectedTs := time.Now().Add(time.Minute * 10).UnixNano()
		if _const.Uint64PointerToBool(ldapModel.Enable) ||
			ldapModel.SyncProtectionTs < time.Now().UnixNano() &&
				Svc.LdapSvc.TryGetLock(
					ldapModel.ID, newProtectedTs, ldapModel.SyncProtectionTs,
				) {

			newGen := ldapModel.SyncGen + 1
			report, err := doCronJob(ldapModel)

			// after success sync from ldap server, mark those user from earlier ldap synchronization
			// as unavailable
			if err != nil {

				_ = Svc.LdapSvc.MarkErrorOccur(ldapModel.ID, ldapModel.SyncGen, err.Error())
			} else {

				report.ActivateSync(ldapModel.Deletes, ldapModel.Inserts, ldapModel.Updates)

				if err := Svc.LdapSvc.UpdateGen(
					ldapModel.ID, newGen,
					report.Entries, report.Inserts, report.Updates, report.Deletes, report.Fails, report.Costs,
				); err != nil {

					log.Error(fmt.Sprintf("Update Ldap gen fail! %s", err))
					return
				}
			}

			Svc.LdapSvc.TryUnLock(ldapModel.ID, newProtectedTs)
		}
	} else {
		if gorm.IsRecordNotFoundError(err) {
			return
		}

		_ = Svc.LdapSvc.MarkErrorOccur(ldapModel.ID, ldapModel.SyncGen, err.Error())
	}
}

func doCronJob(model *model.LdapModel) (*Report, error) {

	report, err := DoSyncFromLDAP(
		context.TODO(), model.Server,
		_const.Uint64PointerToBool(model.Tls), _const.Uint64PointerToBool(model.Md5), model.SyncGen,
		model.BindDn, model.Password,
		model.BaseDn, model.Filter,
		model.AdminBaseDn, model.AdminFilter,
		model.EmailAttr, model.UserNameAttr,
	)

	if err != nil {
		return nil, errors.Wrap(err, "Error when sync from ldap ")
	}

	return report, nil
}

func DoSyncFromLDAP(ctx context.Context, ldapServer string, tls, md5 bool, ldapGen uint64, username, password string,
	baseDN, filter,
	adminBaseDn, adminFilter,
	emailAttr, userNameAttr string) (*Report, error) {
	var report *Report = nil

	err := ldapsrv.ConnectAndThen(
		ldapServer, tls, md5, username, password, func(conn *ldap.Conn) error {

			// >> preparation for search
			if filter == "" {
				filter = "(objectClass=*)"
			}
			if adminFilter == "" {
				adminFilter = "(objectClass=*)"
			}

			adminSet := make(sets.String)
			entriesMapping, err := doSearch(baseDN, filter, emailAttr, userNameAttr, conn)
			if err != nil {
				return err
			}

			if adminBaseDn != "" {
				adminEntriesMapping, err := doSearch(adminBaseDn, adminFilter, emailAttr, userNameAttr, conn)
				if err != nil {
					return err
				}

				for email, entry := range adminEntriesMapping {
					adminSet.Insert(email)
					entriesMapping[email] = entry
				}
			}

			allEntriesCount := len(entriesMapping)

			// >> preparation for sync
			startTs := time.Now()
			updates := 0
			inserts := 0
			success := 0

			nowUser := uint64(0)
			tempList := make([]*model.UserBaseModel, 0)

			// 1): update for exists user
			// if usr profile changed, update directly
			// or else batch update theirs sync gen by tempList
			//
			// 2): create for new user
			// batch create usr by tempList
			//
			// 3): delete for out date user
			// delete those sync gen from elder version
			//
			for {
				list, err := Svc.UserSvc.BatchListByUserId(context.TODO(), nowUser)
				if err != nil {
					log.Error("Error when list user under ldap sync")
					break
				}

				if len(list) == 0 {
					break
				}

				for _, user := range list {
					isAdmin := adminSet.Has(user.Email)

					if entry, ok := entriesMapping[user.Email]; ok {
						usrName := entry.GetAttributeValue(userNameAttr)

						if user.NeedToUpdateProfileInLdap(usrName, entry.DN, isAdmin) {
							if _, err := Svc.UserSvc.UpdateUserByModelWithLdap(
								ctx, user, usrName, entry.DN, ldapGen, isAdmin,
							); err != nil {
								log.Error(
									fmt.Sprintf(
										"Error update user id: %v, dn: %s",
										user.ID, entry.DN,
									),
								)
							}
							success++
							updates++
						} else {
							tempList = append(tempList, user)
						}

						delete(entriesMapping, user.Email)
					}
					if user.ID > nowUser {
						nowUser = user.ID
					}
				}

				if Svc.UserSvc.UpdateUsersLdapGen(tempList, ldapGen) {
					success += len(tempList)
				}
				tempList = make([]*model.UserBaseModel, 0)
			}

			for _, entry := range entriesMapping {

				email := entry.GetAttributeValue(emailAttr)
				if email == "" || !utils.IsEmail(email) {
					continue
				}

				userName := entry.GetAttributeValue(userNameAttr)
				if userName == "" {
					userName = email[:strings.Index(email, "@")]
				}

				isAdmin := adminSet.Has(email)

				tempList = append(
					tempList, &model.UserBaseModel{
						Email:     email,
						Username:  userName,
						Name:      userName,
						LdapDN:    entry.DN,
						LdapGen:   ldapGen,
						IsAdmin:   _const.BoolToUint64Pointer(isAdmin),
						SaName:    model.GenerateSaName(),
						Status:    _const.BoolToUint64Pointer(true),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Uuid:      uuid.NewV4().String(),
						Password:  utils.RandomStr(6),
					},
				)

				if len(tempList) == 100 {
					count := CreateUsers(ctx, tempList)
					inserts += count
					success += count
					tempList = make([]*model.UserBaseModel, 0)
				}
			}

			count := CreateUsers(ctx, tempList)
			inserts += count
			success += count

			if deletes, err := Svc.UserSvc.DeleteOutOfSyncLdapUser(ldapGen); err != nil {
				return errors.Wrap(err, "Error while set out date user status as 0 after ldap sync")
			} else {

				log.Infof(
					"Ldap sync complete %v entries! cost %v, %v user created and %v user updated,"+
						" %v deleted, but %v sync failed.",
					allEntriesCount,
					time.Now().Sub(startTs).String(),
					inserts,
					updates,
					deletes,
					allEntriesCount-inserts-updates,
				)

				report = &Report{
					Entries: allEntriesCount,
					Inserts: inserts,
					Updates: updates,
					Deletes: int(deletes),
					Fails:   allEntriesCount - success,
					Costs:   time.Now().UnixNano() - startTs.UnixNano(),
				}
			}

			return nil
		},
	)

	return report, err
}

func ToMailMapping(es []*ldap.Entry, emailAttr string) map[string]*ldap.Entry {
	m := make(map[string]*ldap.Entry, 0)
	for _, e := range es {
		email := e.GetAttributeValue(emailAttr)
		if email != "" {
			m[email] = e
		}
	}
	return m
}

func CreateUsers(ctx context.Context, list []*model.UserBaseModel) int {
	if Svc.UserSvc.Creates(ctx, list) != nil {

		var dns []string
		for _, user := range list {
			dns = append(dns, user.LdapDN)
		}
		log.Error(fmt.Sprintf("Error when import users by DNs %v", dns))
		return 0
	}

	return len(list)
}

func doSearch(baseDN, filter, emailAttr, userNameAttr string, conn *ldap.Conn) (map[string]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldapsrv.ScopeWholeSubtree, ldapsrv.DerefAlways, 0, 0, false,
		filter,
		[]string{
			"dn", emailAttr, userNameAttr,
		},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, errors.Wrap(err, "Error when sending search request to ldap server ")
	}

	return ToMailMapping(sr.Entries, emailAttr), nil
}

type Report struct {
	Entries int
	Inserts int
	Updates int
	Deletes int
	Fails   int
	Costs   int64
}

func (r *Report) Sum(r1 *Report) {
	r.Entries += r1.Entries
	r.Inserts += r1.Inserts
	r.Updates += r1.Updates
	r.Deletes += r1.Deletes
	r.Fails += r1.Fails
	r.Costs += r1.Costs
}

// We generally only care about valid updates,
// so if these three are 0,
// then the last valid update data is taken
func (r *Report) ActivateSync(deletes, inserts, updates int) {
	if r.Deletes == 0 && r.Inserts == 0 && r.Updates == 0 {
		r.Deletes = deletes
		r.Inserts = inserts
		r.Updates = updates
	}
}
