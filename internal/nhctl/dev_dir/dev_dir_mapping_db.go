package dev_dir

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/yaml.v3"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/dbutils"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strings"
)

var (
	DefaultKey = "DefaultKey"
	splitter   = "-/-"
)

func Update(fun func(dirMapping *DevDirMapping,
// be careful, this map is immutable !!!!
// it is a map generate from #PathToDefaultPackKey and #PackToPath
// when we change #PathToDefaultPackKey or #PackToPath, we should
// regenerate this map
	pathToPack map[DevPath][]*SvcPack) error) error {
	return doGetOrModify(fun, false)
}

func Get(fun func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error) error {
	return doGetOrModify(fun, true)
}

// for test only! it is dangeruas
func ClearAllData() {
	doGetOrModify(
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			dirMapping.PathToDefaultPackKey = map[DevPath]SvcPackKey{}
			dirMapping.PackToPath = map[SvcPackKey]DevPath{}
			return nil
		}, false,
	)
}

// getOrModify true is read only, and false will save the [dirMapping *DevDirMapping] into db
func doGetOrModify(fun func(dirMapping *DevDirMapping,
	pathToPack map[DevPath][]*SvcPack) error,
	readOnly bool) error {
	var path string
	if os.Getenv("TEST") == "" {
		path = nocalhost_path.GetNocalhostDevDirMapping()
	} else {
		path = nocalhost_path.GetTestNocalhostDevDirMapping()
	}

	_ = dbutils.CreateLevelDB(path, false)
	db, err := dbutils.OpenLevelDB(path, readOnly)
	if err != nil {
		_ = db.Close()
		return err
	}
	defer db.Close()

	result := &DevDirMapping{}
	bys, err := db.Get([]byte(DefaultKey))
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			result, err := db.ListAll()
			if err != nil {
				_ = db.Close()
				return err
			}
			for key, val := range result {
				if strings.Contains(key, DefaultKey) {
					bys = []byte(val)
					break
				}
			}
		} else {
			_ = db.Close()
			return errors.Wrap(err, "")
		}
	}
	if len(bys) == 0 {
		result = &DevDirMapping{}
	} else {
		err = yaml.Unmarshal(bys, result)
		if err != nil {
			_ = db.Close()
			return errors.Wrap(err, "")
		}
	}

	if result.PathToDefaultPackKey == nil {
		result.PathToDefaultPackKey = map[DevPath]SvcPackKey{}
	}
	if result.PackToPath == nil {
		result.PackToPath = map[SvcPackKey]DevPath{}
	}
	if result.PackToKubeConfigBytes == nil {
		result.PackToKubeConfigBytes = map[SvcPackKey]string{}
	}
	if result.PackToKubeConfigServer == nil {
		result.PackToKubeConfigServer = map[SvcPackKey]string{}
	}

	if err := fun(result, result.genPathToPackMap()); err != nil {
		return err
	}

	if !readOnly {
		bys, err := yaml.Marshal(result)
		if err != nil {
			return errors.Wrap(err, "")
		}
		return db.Put([]byte(DefaultKey), bys)
	}

	return nil
}

type DevDirMapping struct {
	PathToDefaultPackKey   map[DevPath]SvcPackKey `yaml:"path_to_default_pack_key"`
	PackToPath             map[SvcPackKey]DevPath `yaml:"pack_to_path"`
	PackToKubeConfigBytes  map[SvcPackKey]string  `yaml:"pack_to_kube_config_bytes"`
	PackToKubeConfigServer map[SvcPackKey]string  `yaml:"pack_to_kube_config_server"`
}

// be careful, this map is immutable !!!!
// it is a map generate from #PathToDefaultPackKey and #PackToPath
// when we change #PathToDefaultPackKey or #PackToPath, we should
// regenerate this map
func (devDirMapping *DevDirMapping) genPathToPackMap() map[DevPath][]*SvcPack {
	pathToPack := make(map[DevPath][]*SvcPack, 0)

	for pack, path := range devDirMapping.PackToPath {
		if _, ok := pathToPack[path]; !ok {
			pathToPack[path] = make([]*SvcPack, 0)
		}

		toPack := pack.toPack()
		if toPack == nil {
			log.Logf(fmt.Sprintf("Pack can not case to svcPack %v", pack))
			continue
		}
		pathToPack[path] = append(pathToPack[path], toPack)
	}
	return pathToPack
}

type SvcPackKey string

type DevPath string

func (d DevPath) ToString() string {
	return string(d)
}

type AllSvcPackAssociateByPath struct {
	Packs             map[SvcPackKey]*SvcPack
	Kubeconfigs       map[SvcPackKey]string
	ServerMapping     map[SvcPackKey]string
	DefaultSvcPackKey SvcPackKey
}

func NewSvcPack(ns string,
	nid string,
	app string,
	svcType base.SvcType,
	svc string,
	container string) *SvcPack {
	return &SvcPack{
		Ns:        ns,
		Nid:       nid,
		App:       app,
		SvcType:   svcType,
		Svc:       svc,
		Container: container,
	}
}

type SvcPack struct {
	Ns        string       `yaml:"ns" json:"ns"`
	Nid       string       `yaml:"nid" json:"nid"`
	App       string       `yaml:"app" json:"app"`
	SvcType   base.SvcType `yaml:"svc_type" json:"svc_type"`
	Svc       string       `yaml:"svc" json:"svc"`
	Container string       `yaml:"container" json:"container"`
}

func (svcPackKey *SvcPackKey) toPack() *SvcPack {
	array := strings.Split(string(*svcPackKey), splitter)

	if len(array) < 5 {
		return nil
	}

	// nid is supported later version
	// so need to special compatibility handling
	nid := ""
	if len(array) > 5 {
		nid = array[5]
	}

	return &SvcPack{
		Ns:        array[0],
		Nid:       nid,
		App:       array[1],
		SvcType:   base.SvcType(array[2]),
		Svc:       array[3],
		Container: array[4],
	}
}

func (svcPackKey SvcPackKey) WithoutNid() SvcPackKey {
	str := string(svcPackKey)
	if len(strings.Split(str, splitter)) > 5 {
		return SvcPackKey(str[:strings.LastIndex(str, splitter)])
	}
	return svcPackKey
}

func (svcPack SvcPack) Key() SvcPackKey {
	if svcPack.Container == "" {
		return svcPack.KeyWithoutContainer()
	}

	return SvcPackKey(
		fmt.Sprintf(
			"%s"+splitter+"%s"+splitter+"%s"+splitter+"%s"+splitter+"%s"+splitter+"%s",
			svcPack.Ns, svcPack.App, svcPack.SvcType, svcPack.Svc, svcPack.Container, svcPack.Nid,
		),
	)
}

func (svcPack SvcPack) KeyWithoutContainer() SvcPackKey {
	return SvcPackKey(
		fmt.Sprintf(
			"%s"+splitter+"%s"+splitter+"%s"+splitter+"%s"+splitter+"%s"+splitter+"%s",
			svcPack.Ns, svcPack.App, svcPack.SvcType, svcPack.Svc, "", svcPack.Nid,
		),
	)
}
