package appmeta

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"strings"
)

const (
	SecretType       = "dev.nocalhost/application.meta"
	SecretNamePrefix = "dev.nocalhost.application."

	SecretPreInstallKey = "p"
	SecretManifestKey   = "m"
	SecretDevMetaKey    = "v"
	SecretStateKey      = "s"
	SecretDepKey        = "d"
	SecretConfigKey     = "c"

	UNINSTALLED ApplicationState = "UNINSTALLED"
	INSTALLING  ApplicationState = "INSTALLING"
	INSTALLED   ApplicationState = "INSTALLED"

	DependenceConfigMapPrefix = "nocalhost-depends-do-not-overwrite"
)

func GetApplicationName(secretName string) (string, error) {
	if idx := strings.Index(secretName, "/"); idx > 0 {
		if len(secretName) > idx+1 {
			secretName = secretName[idx+1:]
		}
	}

	if ct := strings.HasPrefix(secretName, SecretNamePrefix); !ct {
		return "", fmt.Errorf("Error while decode Secret, Secret name %s is illegal, must start with %s. ", secretName, SecretNamePrefix)
	}

	return secretName[len(SecretNamePrefix):], nil
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

type ApplicationMeta struct {
	// could not be updated
	Application string `json:"application"`

	// could not be updated
	Ns string `json:"ns"`

	ApplicationState   ApplicationState               `json:"application_state"`
	DepConfigName      string                         `json:"dep_config_name"`
	PreInstallManifest string                         `json:"pre_install_manifest"`
	Manifest           string                         `json:"manifest"`

	// the manifest apply by nhctl apply
	CustomManifest     string                         `json:"custom_manifest"`

	// manage the dev status of the application
	DevMeta            ApplicationDevMeta             `json:"dev_meta"`

	// store all the config of application
	Config             *profile2.NocalHostAppConfigV2 `json:"config"`

	// something like database
	Secret             *corev1.Secret                 `json:"secret"`

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

	bytes, ok := secret.Data[SecretStateKey]
	if !ok {
		return nil, fmt.Errorf("Error while decode Secret, Secret %s is illegal, must contain with data key %s. ", secret.Name, SecretStateKey)
	}

	appMeta := ApplicationMeta{
		Application:      appName,
		Ns:               ns,
		ApplicationState: ApplicationStateOf(string(bytes)),
	}

	if bytes, ok := secret.Data[SecretPreInstallKey]; ok {
		appMeta.PreInstallManifest = string(bytes)
	}

	if bytes, ok := secret.Data[SecretManifestKey]; ok {
		appMeta.Manifest = string(bytes)
	}

	if bytes, ok := secret.Data[SecretDevMetaKey]; ok {
		devMeta := &ApplicationDevMeta{}

		_ = json.Unmarshal(bytes, devMeta)
		appMeta.DevMeta = *devMeta
	}

	if bytes, ok := secret.Data[SecretConfigKey]; ok {
		config := &profile2.NocalHostAppConfigV2{}

		_ = json.Unmarshal(bytes, config)
		appMeta.Config = config
	}

	appMeta.Secret = secret
	return &appMeta, nil
}

func (a *ApplicationMeta) GetApplicationDevMeta() ApplicationDevMeta {
	if a.DevMeta == nil {
		return ApplicationDevMeta{}
	} else {
		return a.DevMeta
	}
}

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
		return errors.Wrap(err, "Error while Initial Application meta ")
	}
	a.Secret = createSecret
	return nil
}

func (a *ApplicationMeta) InitGoClient(kubeConfigPath string) error {
	clientGo, err := clientgoutils.NewClientGoUtils(kubeConfigPath, a.Ns)
	a.clientInner = clientGo
	return err
}

func (a *ApplicationMeta) DeploymentDevStart(deployment, identifier string) error {
	devMeta := a.DevMeta
	if devMeta == nil {
		devMeta = ApplicationDevMeta{}
		a.DevMeta = devMeta
	}

	if _, ok := devMeta[DEPLOYMENT]; !ok {
		devMeta[DEPLOYMENT] = map[ /* resource name */ string] /* identifier */ string{}
	}
	m := devMeta[DEPLOYMENT]

	if _, ok := m[deployment]; ok {
		return errors.New(fmt.Sprintf("Deployment %s is already in DevMode! ", deployment))
	}

	m[deployment] = identifier
	return a.Update()
}

func (a *ApplicationMeta) DeploymentDevEnd(deployment string) error {
	devMeta := a.DevMeta
	if devMeta == nil {
		devMeta = ApplicationDevMeta{}
		a.DevMeta = devMeta
	}

	if _, ok := devMeta[DEPLOYMENT]; !ok {
		devMeta[DEPLOYMENT] = map[ /* resource name */ string] /* identifier */ string{}
	}
	m := devMeta[DEPLOYMENT]

	delete(m, deployment)
	return a.Update()
}

func (a *ApplicationMeta) Update() error {
	a.prepare()

	secret, err := a.clientInner.NameSpace(a.Ns).UpdateSecret(a.Secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "Error while update Application meta ")
	}
	a.Secret = secret
	return nil
}

func (a *ApplicationMeta) prepare() {
	a.Secret.Data[SecretPreInstallKey] = []byte(a.PreInstallManifest)
	a.Secret.Data[SecretManifestKey] = []byte(a.Manifest)
	a.Secret.Data[SecretStateKey] = []byte(a.ApplicationState)
	a.Secret.Data[SecretDepKey] = []byte(a.DepConfigName)

	devMeta, _ := json.Marshal(a.DevMeta)
	a.Secret.Data[SecretDevMetaKey] = devMeta

	config, _ := json.Marshal(a.Config)
	a.Secret.Data[SecretConfigKey] = config
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

func (a *ApplicationMeta) IsHelm() bool {
	return false
}

func (a *ApplicationMeta) Uninstall() error {

	if e := a.cleanUpDepConfigMap(); e != nil {
		log.Error("Error while clean up dep config map %s ", e.Error())
	}

	if a.IsHelm() {
		// todo
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
		if e := a.clientInner.DeleteResourceInfo(info); e != nil {
			log.WarnE(err, fmt.Sprintf("Failed to delete resource %s%s ", info.Name, e.Error()))
		}
	}
}

func (a *ApplicationMeta) cleanPreInstallManifest() {
	resource := clientgoutils.NewResourceFromStr(a.PreInstallManifest)

	//goland:noinspection GoNilness
	infos, err := resource.GetResourceInfo(a.clientInner, true)
	if err != nil {
		log.Error("Error while loading pre install manifest %s, err: %s ", a.PreInstallManifest, err)
	}
	for _, info := range infos {
		if e := a.clientInner.DeleteResourceInfo(info); e != nil {
			log.WarnE(err, fmt.Sprintf("Failed to delete resource %s%s ", info.Name, e.Error()))
		}
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
			err = a.clientInner.DeleteConfigMapByName(cfg.Name)
			if err != nil {
				log.WarnE(err, fmt.Sprintf("Failed to clean up config map: %s", cfg.Name))
			}
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
