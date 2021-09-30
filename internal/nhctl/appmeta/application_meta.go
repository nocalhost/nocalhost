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
	"nocalhost/internal/nhctl/appmeta/secret_operator"
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
	"strings"
	"sync"
)

const (
	SecretType       = "dev.nocalhost/application.meta"
	SecretNamePrefix = "dev.nocalhost.application."
	CmNamePrefix     = "dev.nocalhost.config."
	AnnotationKey    = "dev.nocalhost"
	CmConfigKey      = "config"

	SecretHelmReleaseNameKey = "r"
	SecretPostInstallKey     = "po"
	SecretPostUpgradeKey     = "pou"
	SecretPostDeleteKey      = "pod"
	SecretPreInstallKey      = "p"
	SecretPreUpgradeKey      = "pu"
	SecretPreDeleteKey       = "pd"
	SecretManifestKey        = "m"
	SecretDevMetaKey         = "v"
	SecretAppTypeKey         = "t"
	SecretConfigKey          = "c"
	SecretStateKey           = "s"
	SecretDepKey             = "d"
	SecretNamespaceId        = "nid"

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

	DependenceConfigMapPrefix = "nocalhost-depends-do-not-overwrite"

	DEV_STARTING_SUFFIX = ">...Starting"
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
	nid, _ := utils.GetShortUuid()
	return &ApplicationMeta{
		ApplicationState: INSTALLED,
		Ns:               ns,
		Application:      application,
		DevMeta:          ApplicationDevMeta{},
		Config:           &profile2.NocalHostAppConfigV2{},
		NamespaceId:      nid,
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

	// todo the manifest apply by nhctl apply
	CustomManifest string `json:"custom_manifest"`

	// manage the dev status of the application
	DevMeta ApplicationDevMeta `json:"dev_meta"`

	// store all the config of application
	Config *profile2.NocalHostAppConfigV2 `json:"config"`

	// something like database
	Secret *corev1.Secret `json:"secret"`

	NamespaceId string `json:"namespace_id"`

	// current client go util is injected, may null, be care!
	operator *secret_operator.ClientGoUtilClient
}

func Decode(secret *corev1.Secret) (*ApplicationMeta, error) {
	if secret == nil {
		return nil, fmt.Errorf("Error while decode Secret, Secret is null. ")
	}

	ns := secret.Namespace
	appName, err := GetApplicationName(secret.Name)
	if err != nil {
		return nil, err
	}

	bs, ok := secret.Data[SecretStateKey]
	if !ok {
		return nil, fmt.Errorf(
			"Error while decode Secret, Secret %s is illegal,"+
				" must contain with data key %s. ", secret.Name,
			SecretStateKey,
		)
	}

	appMeta := ApplicationMeta{
		Application:      appName,
		Ns:               ns,
		ApplicationState: ApplicationStateOf(string(bs)),
	}

	if bs, ok := secret.Data[SecretPreInstallKey]; ok {
		appMeta.PreInstallManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPreUpgradeKey]; ok {
		appMeta.PreUpgradeManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPreDeleteKey]; ok {
		appMeta.PreDeleteManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPostInstallKey]; ok {
		appMeta.PostInstallManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPostUpgradeKey]; ok {
		appMeta.PostUpgradeManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretPostDeleteKey]; ok {
		appMeta.PostDeleteManifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretManifestKey]; ok {
		appMeta.Manifest = string(decompress(bs))
	}

	if bs, ok := secret.Data[SecretAppTypeKey]; ok {
		appMeta.ApplicationType = AppType(bs)
	}

	if bs, ok := secret.Data[SecretDevMetaKey]; ok {
		devMeta := &ApplicationDevMeta{}

		_ = yaml.Unmarshal(bs, devMeta)
		appMeta.DevMeta = *devMeta
	}

	if bs, ok := secret.Data[SecretConfigKey]; ok {
		config, _ := unmarshalConfigUnStrict(decompress(bs))
		appMeta.Config = config
	}

	if bs, ok := secret.Data[SecretHelmReleaseNameKey]; ok {
		appMeta.HelmReleaseName = string(bs)
	}

	if bs, ok := secret.Data[SecretNamespaceId]; ok {
		appMeta.NamespaceId = string(bs)
	}

	appMeta.Secret = secret
	return &appMeta, nil
}

func FillingExtField(s *profile2.SvcProfileV2, meta *ApplicationMeta, appName, ns, identifier string) {
	svcType := base.SvcTypeOf(s.GetType())

	devStatus := meta.CheckIfSvcDeveloping(s.GetName(), svcType)

	pack := dev_dir.NewSvcPack(
		ns,
		appName,
		svcType,
		s.GetName(),
		"", // describe can not specify container
	)
	s.Associate = pack.GetAssociatePath().ToString()
	s.Developing = devStatus != NONE
	s.DevelopStatus = string(devStatus)

	if meta.Config != nil {
		svcConfig := meta.Config.GetSvcConfigV2(s.GetName(), svcType)
		if svcConfig != nil {
			s.ServiceConfigV2 = svcConfig
		}
	}

	s.Possess = meta.SvcDevModePossessor(
		s.GetName(), svcType,
		identifier,
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

// Initial initial the application, try to create a secret
// error if create fail
// initial the application will set the state to INSTALLING
func (a *ApplicationMeta) Initial() error {
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
		return err
	}
	a.NamespaceId = id
	createSecret, err := a.operator.Create(a.Ns, &secret)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return err
		}
		return errors.Wrap(err, "Error while Initial Application meta ")
	}
	a.ApplicationState = INSTALLING
	a.Secret = createSecret
	return nil
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
	a.operator = &secret_operator.ClientGoUtilClient{
		ClientInner:     clientGo,
		KubeconfigBytes: content,
	}
	return err
}

func (a *ApplicationMeta) SvcDevModePossessor(name string, svcType base.SvcType, identifier string) bool {
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
func (a *ApplicationMeta) SvcDevStarting(name string, svcType base.SvcType, identifier string) error {
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

func (a *ApplicationMeta) SvcDevStartComplete(name string, svcType base.SvcType, identifier string) error {
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
	return a.Update()
}

func (a *ApplicationMeta) SvcDevEnd(name string, svcType base.SvcType) error {
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

func (a *ApplicationMeta) CheckIfSvcDeveloping(name string, svcType base.SvcType) DevStartStatus {
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

func (a *ApplicationMeta) Update() error {
	return retry.OnError(
		retry.DefaultRetry, func(err error) bool {
			if err != nil {
				if secret, err := a.operator.ReObtainSecret(a.Ns, a.Secret); err != nil {
					return false
				} else {
					a.Secret = secret
					return true
				}
			}
			return false
		}, func() error {
			a.prepare()
			secret, err := a.operator.Update(a.Ns, a.Secret)
			if err != nil {
				return errors.Wrap(err, "Error while update Application meta ")
			}
			a.Secret = secret
			// update daemon application meta manually
			if client, err := daemon_client.NewDaemonClient(false); err == nil {
				_, _ = client.SendUpdateApplicationMetaCommand(
					string(a.operator.GetKubeconfigBytes()), a.Ns, a.Secret.Name, a.Secret,
				)
			}
			return nil
		},
	)
}

func (a *ApplicationMeta) prepare() {
	a.Secret.Data[SecretPreInstallKey] = compress([]byte(a.PreInstallManifest))
	a.Secret.Data[SecretPreUpgradeKey] = compress([]byte(a.PreUpgradeManifest))
	a.Secret.Data[SecretPreDeleteKey] = compress([]byte(a.PreDeleteManifest))
	a.Secret.Data[SecretPostInstallKey] = compress([]byte(a.PostInstallManifest))
	a.Secret.Data[SecretPostUpgradeKey] = compress([]byte(a.PostUpgradeManifest))
	a.Secret.Data[SecretPostDeleteKey] = compress([]byte(a.PostDeleteManifest))
	a.Secret.Data[SecretNamespaceId] = []byte(a.NamespaceId)

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

func (a *ApplicationMeta) IsInstalling() bool {
	return a.ApplicationState == INSTALLING
}

func (a *ApplicationMeta) IsNotInstall() bool {
	return a.ApplicationState == UNINSTALLED
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

		//appProfile, _ := a.GetProfile()
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
	operator := a.operator

	resource := clientgoutils.NewResourceFromStr(a.Manifest)

	//goland:noinspection GoNilness
	infos, err := resource.GetResourceInfo(operator.ClientInner, true)
	if err != nil {
		log.Error("Error while loading manifest %s, err: %s ", a.Manifest, err)
	}
	for _, info := range infos {
		utils.ShouldI(clientgoutils.DeleteResourceInfo(info), "Failed to delete resource "+info.Name)
	}
}

func (a *ApplicationMeta) cleanUpDepConfigMap() error {
	operator := a.operator

	if a.DepConfigName != "" {
		log.Debugf("Cleaning up config map %s", a.DepConfigName)
		err := operator.ClientInner.DeleteConfigMapByName(a.DepConfigName)
		if err != nil {
			return err
		}
		a.DepConfigName = ""
	} else {
		log.Debug("No dependency config map needs to clean up")
	}

	// Clean up all dep config map
	list, err := operator.ClientInner.ListConfigMaps()
	if err != nil {
		return err
	}

	for _, cfg := range list {
		if strings.HasPrefix(cfg.Name, DependenceConfigMapPrefix) {
			utils.ShouldI(
				operator.ClientInner.DeleteConfigMapByName(cfg.Name), "Failed to clean up config map"+cfg.Name,
			)
		}
	}

	return nil
}

// do not call this function direcly
// there should do something in Uninstall
// before Delete Secret
func (a *ApplicationMeta) Delete() error {
	a.Secret.Data[SecretManifestKey] = []byte(a.Manifest)

	name := a.Secret.Name
	err := a.operator.Delete(a.Ns, a.Secret.Name)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrap(err, "Error while Delete Application meta ")
	}
	a.Secret = nil
	// update daemon application meta manually
	if client, err := daemon_client.NewDaemonClient(false); err == nil {
		_, _ = client.SendUpdateApplicationMetaCommand(
			string(a.operator.GetKubeconfigBytes()), a.Ns, name, nil,
		)
	}
	return nil
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
