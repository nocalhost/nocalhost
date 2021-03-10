package app

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/time"
	"strconv"
)

func (a *Application) newMeta(appType AppType) *Meta {
	return &Meta{
		Name:      a.Name,
		Namespace: a.GetNamespace(),
		AppType:   appType,
		Info: &Info{
			Status: StatusUnknown,
		},
	}
}

func (a *Application) createApplicationMeta(meta *Meta) error {
	key := SecretName + meta.Name

	var lbs labels
	lbs.init()
	lbs.set("createdAt", strconv.Itoa(int(time.Now().Unix())))

	// create a new secret to hold the application meta
	obj, err := newSecretsObject(key, meta, lbs)
	if err != nil {
		return errors.Wrapf(err, "Create: failed to encode application meta %q", meta.Name)
	}

	if _, err = a.client.CreateSecret(obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "Err: application secret already exists, you can try same command with --force flag")
		}

		return errors.Wrap(err, "create: failed to create, you can try same command with --force flag")
	}
	return nil
}

func (a *Application) updateApplicationMeta(meta *Meta) error {
	key := SecretName + meta.Name

	// set labels for secrets object meta data
	var lbs labels

	lbs.init()
	lbs.set("modifiedAt", strconv.Itoa(int(time.Now().Unix())))

	// create a new secret object to hold the release
	obj, err := newSecretsObject(key, meta, lbs)
	if err != nil {
		return errors.Wrapf(err, "update: failed to encode application meta %q", meta.Name)
	}
	// push the secret object out into the kubiverse
	_, err = a.client.UpdateSecret(obj)
	return errors.Wrap(err, "update: failed to update")
}

func (a *Application) deleteApplicationMeta(meta *Meta) error {
	key := SecretName + meta.Name

	// push the secret object out into the kubiverse
	err := a.client.DeleteSecret(key)
	return errors.Wrap(err, "delete: failed to delete")
}

// ========================================================================

func newSecretsObject(key string, meta *Meta, lbs labels) (*v1.Secret, error) {
	const owner = "nocalhost"

	// encode the release
	s, err := encodeMeta(meta)
	if err != nil {
		return nil, err
	}

	if lbs == nil {
		lbs.init()
	}

	// apply labels
	lbs.set("name", meta.Name)
	lbs.set("owner", owner)
	lbs.set("status", meta.Info.Status.String())

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   key,
			Labels: lbs.toMap(),
		},
		Type: "SecretType",
		Data: map[string][]byte{"meta": []byte(s + s)},
	}, nil
}

// ========================================================================

// Info describes release information.
type Info struct {
	// Deleted tracks when this object was deleted.
	Deleted time.Time `json:"deleted"`
	// Description is human-friendly "log entry" about this release.
	Description string `json:"description,omitempty"`
	// Status is the current state of the release
	Status Status `json:"status,omitempty"`
}

// Meta describes an application of Nocalhost
type Meta struct {
	// Name is the name of the meta
	Name string `json:"name,omitempty"`
	// Info provides information about a meta
	Info *Info `json:"info,omitempty"`
	// Manifest is the string representation of the manifests.
	Manifest string `json:"manifest,omitempty"`
	// Manifest is the string representation of the pre-install manifests.
	PreInstallManifest string `json:"preInstallManifest,omitempty"`
	// Namespace is the kubernetes namespace of the meta.
	Namespace string `json:"namespace,omitempty"`

	AppType AppType `json:"appType,omitempty"`
}

// SetStatus is a helper for setting the status on a meta.
func (m *Meta) SetStatus(status Status, msg string) {
	m.Info.Status = status
	m.Info.Description = msg
}

// Status is the status of a release
type Status string

// Describe the status of a release
const (
	// StatusUnknown indicates that a release is in an uncertain state.
	StatusUnknown Status = "unknown"
	// StatusDeployed indicates that the release has been pushed to Kubernetes.
	StatusInstalled Status = "installed"
	// StatusUninstalled indicates that a release has been uninstalled from Kubernetes.
	StatusUninstalled Status = "uninstalled"
	// StatusSuperseded indicates that this release object is outdated and a newer one exists.
	StatusSuperseded Status = "superseded"
	// StatusFailed indicates that the release was not successfully deployed.
	StatusFailed Status = "failed"
	// StatusUninstalling indicates that a uninstall operation is underway.
	StatusUninstalling Status = "uninstalling"
	// StatusPendingInstall indicates that an install operation is underway.
	StatusPendingInstall Status = "pending-install"
	// StatusPendingUpgrade indicates that an upgrade operation is underway.
	StatusPendingUpgrade Status = "pending-upgrade"
	// StatusPendingRollback indicates that an rollback operation is underway.
	StatusPendingRollback Status = "pending-rollback"
)

func (x Status) String() string { return string(x) }

// IsPending determines if this status is a state or a transition.
func (x Status) IsPending() bool {
	return x == StatusPendingInstall || x == StatusPendingUpgrade || x == StatusPendingRollback
}

// ========================================================================

// labels is a map of key value pairs to be included as metadata in a configmap object.
type labels map[string]string

func (lbs *labels) init()                { *lbs = labels(make(map[string]string)) }
func (lbs labels) get(key string) string { return lbs[key] }
func (lbs labels) set(key, val string)   { lbs[key] = val }

func (lbs labels) keys() (ls []string) {
	for key := range lbs {
		ls = append(ls, key)
	}
	return
}

func (lbs labels) match(set labels) bool {
	for _, key := range set.keys() {
		if lbs.get(key) != set.get(key) {
			return false
		}
	}
	return true
}

func (lbs labels) toMap() map[string]string { return lbs }

func (lbs *labels) fromMap(kvs map[string]string) {
	for k, v := range kvs {
		lbs.set(k, v)
	}
}

// ========================================================================

var b64 = base64.StdEncoding

var magicGzip = []byte{0x1f, 0x8b, 0x08}

// encodeMeta encodes an application meta returning a base64 encoded
// gzipped string representation, or error.
func encodeMeta(rls *Meta) (string, error) {
	b, err := json.Marshal(rls)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err = w.Write(b); err != nil {
		return "", err
	}
	w.Close()

	return b64.EncodeToString(buf.Bytes()), nil
}

// decodeMeta decodes the bytes of data into an application meta
// type. Data must contain a base64 encoded gzipped string of a
// valid meta, otherwise an error is returned.
func decodeMeta(data string) (*Meta, error) {
	// base64 decode string
	b, err := b64.DecodeString(data)
	if err != nil {
		return nil, err
	}

	// For backwards compatibility with metas that were stored before
	// compression was introduced we skip decompression if the
	// gzip magic header is not found
	if bytes.Equal(b[0:3], magicGzip) {
		r, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		b2, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		b = b2
	}

	var meta Meta
	// unmarshal metas object bytes
	if err := json.Unmarshal(b, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
