/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package appmeta

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/ulikunitz/xz"
	"gopkg.in/yaml.v2"
	"io"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"regexp"
	"strings"
)

const (
	SecretType       = "dev.nocalhost/application.meta"
	SecretNamePrefix = "dev.nocalhost.application."

	SecretHelmReleaseNameKey = "r"
	SecretPreInstallKey      = "p"
	SecretManifestKey        = "m"
	SecretDevMetaKey         = "v"
	SecretAppTypeKey         = "t"
	SecretConfigKey          = "c"
	SecretStateKey           = "s"
	SecretDepKey             = "d"

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
)

var ErrAlreadyDev = errors.New("Svc already in dev mode")

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

// application meta is the application meta info container
type ApplicationMeta struct {
	// could not be updated
	Application string `json:"application"`

	HelmReleaseName string `json:"helm_release_name"`

	// could not be updated
	Ns string `json:"ns"`

	ApplicationType    AppType          `json:"application_type"`
	ApplicationState   ApplicationState `json:"application_state"`
	DepConfigName      string           `json:"dep_config_name"`
	PreInstallManifest string           `json:"pre_install_manifest"`
	Manifest           string           `json:"manifest"`

	// todo the manifest apply by nhctl apply
	CustomManifest string `json:"custom_manifest"`

	// manage the dev status of the application
	DevMeta ApplicationDevMeta `json:"dev_meta"`

	// store all the config of application
	Config *profile2.NocalHostAppConfigV2 `json:"config"`

	// something like database
	Secret *corev1.Secret `json:"secret"`

	// current client go util is injected, may null, be care!
	clientInner *clientgoutils.ClientGoUtils
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

	appMeta.Secret = secret
	return &appMeta, nil
}

func (a *ApplicationMeta) GetClient() *clientgoutils.ClientGoUtils {
	return a.clientInner
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

	createSecret, err := a.clientInner.NameSpace(a.Ns).CreateSecret(&secret, metav1.CreateOptions{})
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
	a.clientInner = clientGo
	return err
}

func (a *ApplicationMeta) SvcDevModePossessor(name string, svcType SvcType, identifier string) bool {
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

func (a *ApplicationMeta) SvcDevStart(name string, svcType SvcType, identifier string) error {
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

func (a *ApplicationMeta) SvcDevEnd(name string, svcType SvcType) error {
	devMeta := a.DevMeta
	if devMeta == nil {
		devMeta = ApplicationDevMeta{}
		a.DevMeta = devMeta
	}

	if _, ok := devMeta[svcType.Alias()]; !ok {
		devMeta[svcType.Alias()] = map[ /* resource name */ string] /* identifier */ string{}
	}
	m := devMeta[svcType.Alias()]

	delete(m, name)
	return a.Update()
}

func (a *ApplicationMeta) CheckIfSvcDeveloping(name string, svcType SvcType) bool {
	devMeta := a.DevMeta
	if devMeta == nil {
		devMeta = ApplicationDevMeta{}
		a.DevMeta = devMeta
	}

	if _, ok := devMeta[svcType.Alias()]; !ok {
		devMeta[svcType.Alias()] = map[ /* resource name */ string] /* identifier */ string{}
	}
	m := devMeta[svcType.Alias()]

	_, ok := m[name]
	return ok
}

func (a *ApplicationMeta) Update() error {
	a.prepare()

	log.Logf("Before update Secret dev meta %+v, app: %s", string(a.Secret.Data[SecretDevMetaKey]), a.Application)
	secret, err := a.clientInner.NameSpace(a.Ns).UpdateSecret(a.Secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "Error while update Application meta ")
	}
	a.Secret = secret
	log.Logf("After update Secret dev meta %+v, app: %s", string(a.Secret.Data[SecretDevMetaKey]), a.Application)
	return nil
}

func (a *ApplicationMeta) prepare() {
	a.Secret.Data[SecretPreInstallKey] = compress([]byte(a.PreInstallManifest))
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
func (a *ApplicationMeta) Uninstall() error {

	if e := a.cleanUpDepConfigMap(); e != nil {
		log.Error("Error while clean up dep config map %s ", e.Error())
	}

	if a.IsHelm() {
		commonParams := make([]string, 0)
		if a.Ns != "" {
			commonParams = append(commonParams, "--namespace", a.Ns)
		}

		//appProfile, _ := a.GetProfile()
		if a.clientInner.KubeConfigFilePath() != "" {
			commonParams = append(commonParams, "--kubeconfig", a.clientInner.KubeConfigFilePath())
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
		); err != nil {
			return err
		}
	}

	// remove pre install manifest
	a.cleanPreInstallManifest()

	// remove manifest
	a.cleanManifest()

	return a.delete()
}

func (a *ApplicationMeta) cleanManifest() {
	resource := clientgoutils.NewResourceFromStr(a.Manifest)

	//goland:noinspection GoNilness
	infos, err := resource.GetResourceInfo(a.clientInner, true)
	if err != nil {
		log.Error("Error while loading manifest %s, err: %s ", a.Manifest, err)
	}
	for _, info := range infos {
		utils.ShouldI(a.clientInner.DeleteResourceInfo(info), "Failed to delete resource "+info.Name)
	}
}

func (a *ApplicationMeta) cleanPreInstallManifest() {
	resource := clientgoutils.NewResourceFromStr(a.PreInstallManifest)

	//goland:noinspection GoNilness
	infos, err := resource.GetResourceInfo(a.clientInner, true)
	utils.ShouldI(err, "Error while loading pre install manifest "+a.PreInstallManifest)

	for _, info := range infos {
		utils.ShouldI(a.clientInner.DeleteResourceInfo(info), "Failed to delete resource "+info.Name)
	}
}

func (a *ApplicationMeta) cleanUpDepConfigMap() error {

	if a.DepConfigName != "" {
		log.Debugf("Cleaning up config map %s", a.DepConfigName)
		err := a.clientInner.DeleteConfigMapByName(a.DepConfigName)
		if err != nil {
			return err
		}
		a.DepConfigName = ""
	} else {
		log.Debug("No dependency config map needs to clean up")
	}

	// Clean up all dep config map
	list, err := a.clientInner.GetConfigMaps()
	if err != nil {
		return err
	}

	for _, cfg := range list {
		if strings.HasPrefix(cfg.Name, DependenceConfigMapPrefix) {
			utils.ShouldI(a.clientInner.DeleteConfigMapByName(cfg.Name), "Failed to clean up config map"+cfg.Name)
		}
	}
	return nil
}

func (a *ApplicationMeta) delete() error {
	a.Secret.Data[SecretManifestKey] = []byte(a.Manifest)

	err := a.clientInner.NameSpace(a.Ns).DeleteSecret(a.Secret.Name)
	if err != nil {
		return errors.Wrap(err, "Error while Delete Application meta ")
	}
	a.Secret = nil
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
