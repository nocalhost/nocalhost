/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package appmeta

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/ulikunitz/xz"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"nocalhost/internal/nhctl/appmeta/operator"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/dev_dir"
	"nocalhost/internal/nhctl/fp"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SecretType       = "dev.nocalhost/application.meta"
	SecretNamePrefix = "dev.nocalhost.application."
	CmNamePrefix     = "dev.nocalhost.config."
	AnnotationKey    = "dev.nocalhost"
	CmConfigKey      = "config"

	SecretUninstallBackOffKey = "time"
	SecretHelmReleaseNameKey  = "r"
	SecretNamespaceIdKey      = "nid"
	SecretPostInstallKey      = "po"
	SecretPostUpgradeKey      = "pou"
	SecretPostDeleteKey       = "pod"
	SecretPreInstallKey       = "p"
	SecretPreUpgradeKey       = "pu"
	SecretPreDeleteKey        = "pd"
	SecretManifestKey         = "m"
	SecretDevMetaKey          = "v"
	SecretAppTypeKey          = "t"
	SecretConfigKey           = "c"
	SecretStateKey            = "s"
	SecretDepKey              = "d"

	Helm           AppType = "helmGit"
	HelmRepo       AppType = "helmRepo"
	Manifest       AppType = "rawManifest"
	ManifestGit    AppType = "rawManifestGit"
	ManifestLocal  AppType = "rawManifestLocal"
	HelmLocal      AppType = "helmLocal"
	KustomizeGit   AppType = "kustomizeGit"
	KustomizeLocal AppType = "kustomizeLocal"

	UNINSTALLED ApplicationState = "UNINSTALLED"
	INSTALLING  ApplicationState = "INSTALLING"
	INSTALLED   ApplicationState = "INSTALLED"

	UNKNOWN ApplicationState = "UNKNOWN"

	DependenceConfigMapPrefix = "nocalhost-depends-do-not-overwrite"

	DEV_STARTING_SUFFIX = ">...Starting"
	DuplicateSuffix     = "-duplicate"
)

var ErrAlreadyDev = errors.New("Svc already in dev mode")

func GetAppNameFromConfigMapName(cmName string) string {
	if HasConfigMapPrefix(cmName) {
		return strings.TrimPrefix(cmName, CmNamePrefix)
	}
	return ""
}

func HasConfigMapPrefix(key string) bool {
	return strings.HasPrefix(key, CmNamePrefix)
}

func ConfigMapName(appName string) string {
	return CmNamePrefix + appName
}

// resolve Application name by k8s 'metadata.name'
func GetApplicationName(secretName string) (string, error) {
	if idx := strings.Index(secretName, "/"); idx > 0 {
		if len(secretName) > idx+1 {
			secretName = secretName[idx+1:]
		}
	}

	if ct := strings.HasPrefix(secretName, SecretNamePrefix); !ct {
		return "", fmt.Errorf(
			"Error while decode Secret, Secret name %s is illegal,"+
				" must start with %s. ", secretName, SecretNamePrefix,
		)
	}

	return secretName[len(SecretNamePrefix):], nil
}

type AppType string

func AppTypeOf(s string) AppType {
	switch s {
	case string(Helm):
		return Helm
	case string(HelmRepo):
		return HelmRepo
	case string(HelmLocal):
		return HelmLocal
	case string(Manifest):
		return Manifest
	case string(ManifestGit):
		return ManifestGit
	case string(ManifestLocal):
		return ManifestLocal
	case string(KustomizeGit):
		return KustomizeGit
	default:
		return Manifest
	}
}

func (a AppType) IsHelm() bool {
	return a == Helm || a == HelmRepo || a == HelmLocal
}

type ApplicationState string

func ApplicationStateOf(s string) ApplicationState {
	switch s {
	case string(INSTALLING):
		return INSTALLING
	case string(INSTALLED):
		return INSTALLED
	case string(UNKNOWN):
		return UNKNOWN
	}
	return UNINSTALLED
}

type ApplicationMetas []*ApplicationMeta
type ApplicationMetaSimples []*ApplicationMetaSimple

// describe the applications meta for output
func (as ApplicationMetas) Desc() (result ApplicationMetaSimples) {
	for _, meta := range as {
		result = append(
			result, &ApplicationMetaSimple{
				Application:        meta.Application,
				Ns:                 meta.Ns,
				ApplicationState:   meta.ApplicationState,
				DevMeta:            meta.DevMeta,
				Manifest:           meta.Manifest,
				PreInstallManifest: meta.PreInstallManifest,
			},
		)
	}
	return result
}

type ApplicationMetaSimple struct {
	Application      string           `json:"application"`
	Ns               string           `json:"ns"`
	ApplicationState ApplicationState `json:"application_state"`
	// manage the dev status of the application
	DevMeta            ApplicationDevMeta `json:"dev_meta"`
	Manifest           string             `json:"manifest"`
	PreInstallManifest string             `json:"pre_install_manifest"`
}

func FakeAppMeta(ns, application string) *ApplicationMeta {
	return &ApplicationMeta{
		ApplicationState: INSTALLED,
		Ns:               ns,
		Application:      application,
		DevMeta:          ApplicationDevMeta{},
		Config:           &profile2.NocalHostAppConfigV2{Migrated: true},
		NamespaceId:      "fakedNid",
	}
}

// application meta is the application meta info container
type ApplicationMeta struct {
	locker sync.Mutex

	// could not be updated
	Application string `json:"application"`

	HelmReleaseName string `json:"helm_release_name"`

	// could not be updated
	Ns string `json:"ns"`

	ApplicationType  AppType          `json:"application_type"`
	ApplicationState ApplicationState `json:"application_state"`
	DepConfigName    string           `json:"dep_config_name"`

	PreInstallManifest  string `json:"pre_install_manifest"`
	PostInstallManifest string `json:"post_install_manifest"`
	PreUpgradeManifest  string `json:"pre_upgrade_manifest"`
	PostUpgradeManifest string `json:"post_upgrade_manifest"`
	PreDeleteManifest   string `json:"pre_delete_manifest"`
	PostDeleteManifest  string `json:"post_delete_manifest"`

	Manifest string `json:"manifest"`

	// back off timestamp when application uninstall
	// may invalid while inaccurate sys time
	UninstallBackOff int64 `json:"uninstall_back_off"`

	// manage the dev status of the application
	DevMeta ApplicationDevMeta `json:"dev_meta"`

	// store all the config of application
	Config *profile2.NocalHostAppConfigV2 `json:"config"`

	// something like database
	Secret *corev1.Secret `json:"secret"`

	// to distinguish same ns/app when using multiple K8s cluster
	NamespaceId string `json:"namespace_id"`

	// current client go util is injected, may null, be care!
	operator *operator.ClientGoUtilClient
}

func (a *ApplicationMeta) ReAssignmentBySecret(secret *corev1.Secret) error {
	a.Secret = secret

	if secret == nil {
		return fmt.Errorf("Error while decode Secret, Secret is null. ")
	}

	ns := secret.Namespace
	appName, err := GetApplicationName(secret.Name)
	if err != nil {
		return err
	}

	bs, ok := secret.Data[SecretStateKey]
	if !ok {
		return fmt.Errorf(
			"Error while decode Secret, Secret %s is illegal,"+
				" must contain with data key %s. ", secret.Name,
			SecretStateKey,
		)
	}

	a.Application = appName
	a.Ns = ns
	a.ApplicationState = ApplicationStateOf(string(bs))

	if bs, ok := secret.Data[SecretUninstallBackOffKey]; ok {
		str := string(bs)
		valInt, err := strconv.Atoi(str)
		if err != nil {
			a.UninstallBackOff = 0
		} else {
			a.UninstallBackOff = int64(valInt)
		}
	}

	if bs, ok := secret.Data[SecretPreInstallKey]; ok {
		a.PreInstallManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPreUpgradeKey]; ok {
		a.PreUpgradeManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPreDeleteKey]; ok {
		a.PreDeleteManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPostInstallKey]; ok {
		a.PostInstallManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPostUpgradeKey]; ok {
		a.PostUpgradeManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPostDeleteKey]; ok {
		a.PostDeleteManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretManifestKey]; ok {
		a.Manifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretAppTypeKey]; ok {
		a.ApplicationType = AppType(bs)
	}

	if bs, ok := secret.Data[SecretDevMetaKey]; ok {
		devMeta := &ApplicationDevMeta{}

		_ = yaml.Unmarshal(bs, devMeta)
		a.DevMeta = *devMeta
	}

	if bs, ok := secret.Data[SecretConfigKey]; ok {
		config, _ := unmarshalConfigUnStrict(decompress(bs))
		a.Config = config
	}

	if a.Config == nil {
		a.Config = &profile2.NocalHostAppConfigV2{}
	}

	if bs, ok := secret.Data[SecretHelmReleaseNameKey]; ok {
		a.HelmReleaseName = string(bs)
	}

	if bs, ok := secret.Data[SecretNamespaceIdKey]; ok {
		a.NamespaceId = string(bs)
	}

	return nil
}

func Decode(secret *corev1.Secret) (*ApplicationMeta, error) {
	appMeta := &ApplicationMeta{}

	if err := appMeta.ReAssignmentBySecret(secret); err != nil {
		return nil, err
	}

	return appMeta, nil
}

func FillingExtField(s *profile2.SvcProfileV2, meta *ApplicationMeta, appName, ns, identifier string) {
	svcType := base.SvcType(s.GetType())

	s.DevModeType = meta.GetCurrentDevModeTypeOfWorkload(s.GetName(), base.SvcType(s.GetType()), identifier)
	devStatus := meta.CheckIfSvcDeveloping(s.GetName(), identifier, svcType, s.DevModeType)

	pack := dev_dir.NewSvcPack(
		ns,
		meta.NamespaceId,
		appName,
		svcType,
		s.GetName(),
		"", // describe can not specify container
	)

	// associate
	s.Associate = pack.GetAssociatePathCache().ToString()
	s.Developing = devStatus != NONE
	s.DevelopStatus = string(devStatus)

	//if meta.Config != nil {
	//	svcConfig := meta.Config.GetSvcConfigV2(s.GetName(), svcType)
	//	if svcConfig != nil {
	//		s.ServiceConfigV2 = svcConfig
	//	}
	//}

	s.Possess = meta.SvcDevModePossessor(
		s.GetName(), svcType,
		identifier, s.DevModeType,
	)
}

// sometimes meta will not initail the go client, this method
// can present to use a temp kubeconfig for some K8s's oper
func (a *ApplicationMeta) DoWithTempOperator(configBytes []byte, funny func() error) error {
	randomKubeconfigFile := fp.NewRandomTempPath().MkdirThen().RelOrAbs("tmpkubeconfig")
	if err := randomKubeconfigFile.WriteFile(string(configBytes)); err != nil {
		return errors.Wrap(err, "Error when gen tempKubeconfig while initialize temp operator for meta ")
	}

	return a.doMutex(
		func() error {
			backup := a.operator
			defer func() {
				a.operator = backup
				_ = randomKubeconfigFile.Doom()
			}()

			if err := a.InitGoClient(randomKubeconfigFile.Abs()); err != nil {
				return err
			}

			return funny()
		},
	)
}

func (a *ApplicationMeta) doMutex(funny func() error) error {
	defer a.locker.Unlock()
	a.locker.Lock()
	return funny()
}

func (a *ApplicationMeta) GenerateNidINE() error {
	if a.NamespaceId == "" {
		id, err := utils.GetShortUuid()
		if err != nil {
			return err
		}
		a.NamespaceId = id
		return a.Update()
	}
	return nil
}

func (a *ApplicationMeta) GetApplicationConfig() *profile2.ApplicationConfig {
	if a == nil || a.Config == nil {
		return &profile2.ApplicationConfig{}
	}

	return &a.Config.ApplicationConfig
}

func (a *ApplicationMeta) GetClient() *clientgoutils.ClientGoUtils {
	return a.operator.ClientInner
}

func (a *ApplicationMeta) GetApplicationDevMeta() ApplicationDevMeta {
	if a.DevMeta == nil {
		return ApplicationDevMeta{}
	} else {
		return a.DevMeta
	}
}

func (a *ApplicationMeta) initialFreshSecret() (*corev1.Secret, error) {
	b := false
	m := map[string][]byte{}
	m[SecretStateKey] = []byte(INSTALLING)

	secret := corev1.Secret{
		Immutable: &b,
		Data:      m,
		Type:      SecretType,
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretNamePrefix + a.Application,
			Namespace: a.Ns,
		},
	}
	id, err := utils.GetShortUuid()
	if err != nil {
		return nil, err
	}
	a.NamespaceId = id
	return &secret, nil
}

// Initial initial the application,
// try to create a secret if not exist and it's state is INSTALLING
// or modify the exists secret as INSTALLING if the secret state is UNINSTALL
// params: force - force to re initial the appmeta, (while app uninstall, there is 10s re initial protect)
// if want to disable the protect, set force as true
func (a *ApplicationMeta) Initial(force bool) error {
	return a.doPreCheckThen(
		func(exists bool) error {
			if exists {
				a.ApplicationState = INSTALLING
				return a.Update()
			} else {
				if secret, err := a.initialFreshSecret(); err != nil {
					return err
				} else {

					createSecret, err := a.operator.Create(a.Ns, secret)
					if err != nil {
						if k8serrors.IsAlreadyExists(err) {
							return err
						}
						return errors.Wrap(err, "Error while Initial Application meta, fail to create secret ")
					}

					if err := a.ReAssignmentBySecret(createSecret); err != nil {
						return errors.Wrap(err, "Error while Initial Application meta, fail to decode secret ")
					}
				}
				return nil
			}
		}, force,
	)
}

// Initial initial the application,
// try to create a secret if not exist and it's state is INSTALLED
// or modify the exists secret as INSTALLED if the secret state is UNINSTALL
func (a *ApplicationMeta) OneTimesInitial(fun func(meta *ApplicationMeta), force bool) error {

	return a.doPreCheckThen(
		func(exists bool) error {
			if exists {
				a.ApplicationState = INSTALLED

				fun(a)

				return a.Update()
			} else {
				if secret, err := a.initialFreshSecret(); err != nil {
					return err
				} else {
					a.ApplicationState = INSTALLED
					a.Secret = secret

					fun(a)

					a.prepare()

					createSecret, err := a.operator.Create(a.Ns, secret)
					if err != nil {
						if k8serrors.IsAlreadyExists(err) {
							return err
						}
						return errors.Wrap(err, "Error while Initial Application meta, fail to create secret ")
					}

					if err := a.ReAssignmentBySecret(createSecret); err != nil {
						return errors.Wrap(err, "Error while Initial Application meta, fail to decode secret ")
					}
				}
				return nil
			}
		}, force,
	)
}

// do some pre check of initial a secret
func (a *ApplicationMeta) doPreCheckThen(howToInitialMeta func(bool) error, skipProtectedCheck bool) error {
	exists := true
	get, err := a.operator.Get(a.Ns, SecretNamePrefix+a.Application)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			exists = false
		} else {
			return errors.Wrap(
				err, fmt.Sprintf(
					"Error while Initial Application meta, fail to get secret , error: %s\n",
					err.Error(),
				),
			)
		}
	}

	// if the secret already exist, and it's install status
	// is UNINSTALL, make it as installing otherwise return
	// an error because other process may done this
	//
	if exists {

		if err := a.ReAssignmentBySecret(get); err != nil {
			return errors.Wrap(err, "Error while Initial Application meta, fail to decode secret ")
		}

		// if secret already exist, and we want to re init it
		// we should check it's protect from UninstallBackOff
		// after it last uninstall
		//
		// and we can set skipProtectedCheck as true to skip
		// this check
		if skipProtectedCheck {

		} else if a.IsNotInstall() {

			if a.ProtectedFromReInstall() {
				return errors.New(
					"Application may uninstalling, " +
						"the secret is protected to re initial, please try again later ",
				)
			}
		}
	}

	return howToInitialMeta(exists)
}

func (a *ApplicationMeta) InitGoClient(kubeConfigPath string) error {
	clientGo, err := clientgoutils.NewClientGoUtils(kubeConfigPath, a.Ns)
	if err != nil {
		return err
	}

	content, err := ioutil.ReadFile(kubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "can not read kubeconfig content, path: "+kubeConfigPath)
	}
	a.operator = &operator.ClientGoUtilClient{
		ClientInner:     clientGo,
		KubeconfigBytes: content,
		Dc:              clientGo.GetDynamicClient(),
	}
	return err
}

func (a *ApplicationMeta) SvcDevModePossessor(name string, svcType base.SvcType, identifier string, modeType profile2.DevModeType) bool {
	name = devModeName(name, identifier, modeType)
	devMeta := a.DevMeta
	if devMeta == nil {
		devMeta = ApplicationDevMeta{}
		a.DevMeta = devMeta
	}

	if _, ok := devMeta[svcType.Alias()]; !ok {
		devMeta[svcType.Alias()] = map[ /* resource name */ string] /* identifier */ string{}
	}
	m := devMeta[svcType.Alias()]
	return m[name] == identifier && identifier != ""
}

// SvcDevStarting call this func first recode 'name>...starting' as developing
// while complete enter dev start, should call #SvcDevStartComplete to mark svc completely enter dev mode
func (a *ApplicationMeta) SvcDevStarting(name string, svcType base.SvcType, identifier string, modeType profile2.DevModeType) error {
	name = devModeName(name, identifier, modeType)
	devMeta := a.DevMeta
	if devMeta == nil {
		devMeta = ApplicationDevMeta{}
		a.DevMeta = devMeta
	}

	if _, ok := devMeta[svcType.Alias()]; !ok {
		devMeta[svcType.Alias()] = map[ /* resource name */ string] /* identifier */ string{}
	}
	m := devMeta[svcType.Alias()]

	if _, ok := m[name]; ok {
		return ErrAlreadyDev
	}

	inDevStartingMark := devStartMarkSign(name)
	if _, ok := m[inDevStartingMark]; ok {
		return ErrAlreadyDev
	}

	m[inDevStartingMark] = identifier
	return a.Update()
}

func HasDevStartingSuffix(name string) bool {
	return strings.HasSuffix(name, DEV_STARTING_SUFFIX)
}

func devStartMarkSign(name string) string {
	return fmt.Sprintf("%s%s", name, DEV_STARTING_SUFFIX)
}

func devModeName(name, identifier string, modeType profile2.DevModeType) string {
	if !modeType.IsReplaceDevMode() {
		return name + "-" + string(modeType) + "-" + identifier
	}
	return name
}

func (a *ApplicationMeta) SvcDevStartComplete(name string, svcType base.SvcType, identifier string, modeType profile2.DevModeType) error {
	name = devModeName(name, identifier, modeType)
	devMeta := a.DevMeta
	if devMeta == nil {
		devMeta = ApplicationDevMeta{}
		a.DevMeta = devMeta
	}

	if _, ok := devMeta[svcType.Alias()]; !ok {
		devMeta[svcType.Alias()] = map[ /* resource name */ string] /* identifier */ string{}
	}
	m := devMeta[svcType.Alias()]

	if _, ok := m[name]; ok {
		return ErrAlreadyDev
	}

	m[name] = identifier
	inDevStartingMark := devStartMarkSign(name)
	delete(m, inDevStartingMark)
	return a.Update()
}

func (a *ApplicationMeta) SvcDevEnd(name, identifier string, svcType base.SvcType, modeType profile2.DevModeType) error {
	name = devModeName(name, identifier, modeType)
	devMeta := a.DevMeta
	if devMeta == nil {
		devMeta = ApplicationDevMeta{}
		a.DevMeta = devMeta
	}

	if _, ok := devMeta[svcType.Alias()]; !ok {
		devMeta[svcType.Alias()] = map[ /* resource name */ string] /* identifier */ string{}
	}
	m := devMeta[svcType.Alias()]

	inDevStartingMark := devStartMarkSign(name)

	delete(m, inDevStartingMark)
	delete(m, name)
	return a.Update()
}

func (a *ApplicationMeta) CheckIfSvcDeveloping(name, identifier string, svcType base.SvcType, modeType profile2.DevModeType) DevStartStatus {
	name = devModeName(name, identifier, modeType)
	devMeta := a.DevMeta
	if devMeta == nil {
		devMeta = ApplicationDevMeta{}
		a.DevMeta = devMeta
	}

	if _, ok := devMeta[svcType.Alias()]; !ok {
		devMeta[svcType.Alias()] = map[ /* resource name */ string] /* identifier */ string{}
	}
	m := devMeta[svcType.Alias()]

	if _, ok := m[name]; ok {
		return STARTED
	}

	if _, ok := m[devStartMarkSign(name)]; ok {
		return STARTING
	}

	return NONE
}

// Update if update occurs errors, it will get a new secret from k8s, and retry update again, default retry times is 5
func (a *ApplicationMeta) Update() error {
	return retry.OnError(
		retry.DefaultRetry, func(err error) bool {
			if err != nil {
				if secret, _ := a.operator.Get(a.Ns, SecretNamePrefix+a.Application); secret != nil {
					a.Secret = secret
				}
				return true
			}
			return false
		}, func() error {
			if a.Secret == nil {
				return errors.New("secret not found")
			}
			if a.Secret.Data == nil {
				return errors.New("secret data is nil")
			}
			a.prepare()
			secret, err := a.operator.Update(a.Ns, a.Secret)
			if err != nil {
				return errors.Wrap(err, "Error while update Application meta ")
			}
			a.Secret = secret
			// update daemon application meta manually
			if client, err := daemon_client.GetDaemonClient(false); err == nil {
				_, _ = client.SendUpdateApplicationMetaCommand(
					string(a.operator.GetKubeconfigBytes()), a.Ns, a.Secret.Name, a.Secret,
				)
			}
			return nil
		},
	)
}

func (a *ApplicationMeta) prepare() {
	a.Secret.Data[SecretUninstallBackOffKey] = []byte(strconv.FormatInt(a.UninstallBackOff, 10))
	a.Secret.Data[SecretPreInstallKey] = compress([]byte(a.PreInstallManifest))
	a.Secret.Data[SecretPreUpgradeKey] = compress([]byte(a.PreUpgradeManifest))
	a.Secret.Data[SecretPreDeleteKey] = compress([]byte(a.PreDeleteManifest))
	a.Secret.Data[SecretPostInstallKey] = compress([]byte(a.PostInstallManifest))
	a.Secret.Data[SecretPostUpgradeKey] = compress([]byte(a.PostUpgradeManifest))
	a.Secret.Data[SecretPostDeleteKey] = compress([]byte(a.PostDeleteManifest))
	a.Secret.Data[SecretNamespaceIdKey] = []byte(a.NamespaceId)

	a.Secret.Data[SecretManifestKey] = compress([]byte(a.Manifest))

	config, _ := yaml.Marshal(a.Config)
	a.Secret.Data[SecretConfigKey] = compress(config)

	a.Secret.Data[SecretStateKey] = []byte(a.ApplicationState)
	a.Secret.Data[SecretDepKey] = []byte(a.DepConfigName)
	a.Secret.Data[SecretAppTypeKey] = []byte(a.ApplicationType)
	a.Secret.Data[SecretHelmReleaseNameKey] = []byte(a.HelmReleaseName)

	devMeta, _ := yaml.Marshal(&a.DevMeta)
	a.Secret.Data[SecretDevMetaKey] = devMeta
}

func (a *ApplicationMeta) IsInstalled() bool {
	return a.ApplicationState == INSTALLED
}

func (a *ApplicationMeta) IsUnknown() bool {
	return a.ApplicationState == UNKNOWN
}

func (a *ApplicationMeta) IsInstalling() bool {
	return a.ApplicationState == INSTALLING
}

func (a *ApplicationMeta) IsNotInstall() bool {
	return a.ApplicationState == UNINSTALLED
}

func (a *ApplicationMeta) ProtectedFromReInstall() bool {
	return a.UninstallBackOff > time.Now().UnixNano()
}

func (a *ApplicationMeta) NotInstallTips() string {
	return fmt.Sprintf(
		"Application %s in ns %s is not installed or under installing, "+
			"or maybe the kubeconfig provided has not permitted to this namespace ",
		a.Application, a.Ns,
	)
}

func (a *ApplicationMeta) IsHelm() bool {
	return a.ApplicationType == Helm || a.ApplicationType == HelmRepo || a.ApplicationType == HelmLocal
}

// Uninstall uninstall the application and delete the secret from k8s cluster
func (a *ApplicationMeta) Uninstall(force bool) error {
	preDelete := a.PreDeleteManifest
	postDelete := a.PostDeleteManifest

	if preDelete != "" {
		log.Info("Executing pre-uninstall hook")
		if err := a.operator.ExecHook(a.Application, a.Ns, preDelete); err != nil {
			return errors.Wrap(err, "Error while exec pre-delete hook ")
		}
	}

	if e := a.cleanUpDepConfigMap(); e != nil {
		log.Error("Error while clean up dep config map %s ", e.Error())
	}

	// remove hook
	a.operator.CleanManifest(a.PreInstallManifest)
	a.operator.CleanManifest(a.PostInstallManifest)
	a.operator.CleanManifest(a.PreUpgradeManifest)
	a.operator.CleanManifest(a.PostUpgradeManifest)
	a.operator.CleanManifest(a.PreDeleteManifest)

	// remove manifest
	a.cleanManifest()

	if a.IsHelm() {
		commonParams := make([]string, 0)
		if a.Ns != "" {
			commonParams = append(commonParams, "--namespace", a.Ns)
		}

		if a.operator.ClientInner.KubeConfigFilePath() != "" {
			commonParams = append(commonParams, "--kubeconfig", a.operator.ClientInner.KubeConfigFilePath())
		}

		uninstallParams := []string{"uninstall"}

		if a.HelmReleaseName != "" {
			uninstallParams = append(uninstallParams, a.HelmReleaseName)
		} else {
			uninstallParams = append(uninstallParams, a.Application)
		}

		uninstallParams = append(uninstallParams, commonParams...)
		if _, err := tools.ExecCommand(
			nil, true, true,
			true, "helm", uninstallParams...,
		); err != nil && !force {
			return err
		}
	}

	a.operator.CleanCustomResource(a.Application, a.Ns)

	if err := a.Delete(); err != nil {
		return err
	}

	if postDelete != "" {
		log.Info("Executing post-uninstall hook")
		if err := a.operator.ExecHook(a.Application, a.Ns, postDelete); err != nil {
			return errors.Wrap(err, "Error while exec post-delete hook ")
		}
	}
	a.operator.CleanManifest(a.PostDeleteManifest)
	return nil
}

func (a *ApplicationMeta) cleanManifest() {
	op := a.operator

	resource := clientgoutils.NewResourceFromStr(a.Manifest)

	//goland:noinspection GoNilness
	infos, err := resource.GetResourceInfo(op.ClientInner, true)
	if err != nil {
		log.Error("Error while loading manifest %s, err: %s ", a.Manifest, err)
	}
	for _, info := range infos {
		utils.ShouldI(clientgoutils.DeleteResourceInfo(info), "Failed to delete resource "+info.Name)
	}
}

func (a *ApplicationMeta) cleanUpDepConfigMap() error {
	op := a.operator

	if a.DepConfigName != "" {
		log.Debugf("Cleaning up config map %s", a.DepConfigName)
		err := op.ClientInner.DeleteConfigMapByName(a.DepConfigName)
		if err != nil {
			return err
		}
		a.DepConfigName = ""
	} else {
		log.Debug("No dependency config map needs to clean up")
	}

	// Clean up all dep config map
	list, err := op.ClientInner.ListConfigMaps()
	if err != nil {
		return err
	}

	for _, cfg := range list {
		if strings.HasPrefix(cfg.Name, DependenceConfigMapPrefix) {
			utils.ShouldI(
				op.ClientInner.DeleteConfigMapByName(cfg.Name), "Failed to clean up config map"+cfg.Name,
			)
		}
	}

	return nil
}

// Delete a application will not actually delete the secret
// just make the application as UNINSTALL state
// and set a protect time UninstallBackOff
// then clean the secret data
func (a *ApplicationMeta) Delete() error {
	a.HelmReleaseName = ""
	a.ApplicationType = ""
	a.ApplicationState = UNINSTALLED
	a.DepConfigName = ""
	a.PreInstallManifest = ""
	a.PostInstallManifest = ""
	a.PreUpgradeManifest = ""
	a.PostUpgradeManifest = ""
	a.PreDeleteManifest = ""
	a.PostDeleteManifest = ""
	a.Manifest = ""
	a.DevMeta = map[base.SvcType]map[string]string{}
	a.UninstallBackOff = time.Now().Add(time.Second * 10).UnixNano()

	return a.Update()
}

func (a *ApplicationMeta) GetCurrentDevModeTypeOfWorkload(workloadName string, workloadType base.SvcType, identifier string) profile2.DevModeType {
	if a.CheckIfSvcDeveloping(workloadName, identifier, workloadType, profile2.DuplicateDevMode) != NONE {
		return profile2.DuplicateDevMode
	}
	if a.CheckIfSvcDeveloping(workloadName, identifier, workloadType, profile2.ReplaceDevMode) != NONE {
		return profile2.ReplaceDevMode
	}
	return profile2.NoneDevMode
}

func (a *ApplicationMeta) NewResourceReader() *clientgoutils.Resource {
	return clientgoutils.NewResourceFromStr(a.Manifest)
}

func compress(input []byte) []byte {
	var buf bytes.Buffer
	// compress text
	w, err := xz.NewWriter(&buf)
	if err != nil {
		log.Errorf("xz.NewWriter error %s", err)
		return nil
	}
	if _, err := io.WriteString(w, string(input)); err != nil {
		log.Errorf("WriteString error %s", err)
		return nil
	}
	if err := w.Close(); err != nil {
		log.Errorf("w.Close error %s", err)
		return nil
	}

	return buf.Bytes()
}

func decompress(input []byte) []byte {
	if input == nil || len(input) == 0 {
		return input
	}

	var buf bytes.Buffer

	buf.Write(input)
	// decompress buffer and write output to stdout
	r, err := xz.NewReader(&buf)
	if err != nil {
		log.Errorf("NewReader error %s", err)
		return nil
	}

	buffer := new(bytes.Buffer)
	_, _ = buffer.ReadFrom(r)
	return buffer.Bytes()
}

func unmarshalConfigUnStrict(cfg []byte) (*profile2.NocalHostAppConfigV2, error) {
	result := &profile2.NocalHostAppConfigV2{}
	err := yaml.Unmarshal(cfg, result)
	if err != nil {
		re, _ := regexp.Compile("remoteDebugPort: \"[0-9]*\"") // fix string convert int error
		rep := re.ReplaceAllString(string(cfg), "")
		err = yaml.Unmarshal([]byte(rep), result)
	}
	return result, err
}
