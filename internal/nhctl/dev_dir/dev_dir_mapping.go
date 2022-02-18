package dev_dir

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"strings"
	"sync"
	"time"
)

var (
	NO_DEFAULT_PACK = errors.New("Current Svc pack not found ")
	cacheLock       = sync.Mutex{}
	cache           *DevDirMapping
)

func Initial() {
	FlushCache()
	tick := time.NewTicker(time.Second * 5)
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("Fail to flush dir-svc cache in cacher ", r)
					}
				}()
				<-tick.C
				FlushCache()
			}()
		}
	}()
}

func FlushCache() {
	if err := Get(
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			cacheLock.Lock()
			defer cacheLock.Unlock()
			cache = dirMapping
			return nil
		},
	); err != nil {
		log.ErrorE(err, fmt.Sprintf("Fail to flush dir-svc cache"))
	}
}

func (svcPack *SvcPack) GetAssociatePathCache() DevPath {
	if !svcPack.valid() {
		log.Logf("Current svc is invalid to get associate path, %v", svcPack)
		return ""
	}

	if cache == nil {
		FlushCache()
	}

	cacheLock.Lock()
	defer cacheLock.Unlock()

	var path DevPath
	if _, ok := cache.PackToPath[svcPack.Key()]; ok {
		path = cache.PackToPath[svcPack.Key()]
	} else {
		path = cache.PackToPath[svcPack.KeyWithoutContainer()]
	}
	return path
}

// get associate path of svcPack
// if no path match, try with svc with none container
func (svcPack *SvcPack) GetAssociatePath() DevPath {
	if !svcPack.valid() {
		log.Logf("Current svc is invalid to get associate path, %v", svcPack)
		return ""
	}

	var path DevPath
	if err := Get(
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {

			// first try to get from full key

			if p, ok := dirMapping.PackToPath[svcPack.Key()]; ok {
				path = p
				return nil
			} else if p, ok := dirMapping.PackToPath[svcPack.KeyWithoutContainer()]; ok {
				path = p
				return nil
			}

			// them remove the nid and try again

			if p, ok := dirMapping.PackToPath[svcPack.Key().WithoutNid()]; ok {
				path = p
			} else {
				path = dirMapping.PackToPath[svcPack.KeyWithoutContainer().WithoutNid()]
			}
			return nil
		},
	); err != nil {
		log.ErrorE(err, fmt.Sprintf("Current svc is fail to get associate path, %v", svcPack))
		return ""
	}
	return path
}

// return "" if error occur
func (svcPack *SvcPack) GetKubeConfigBytesAndServer() (string, string) {
	if !svcPack.valid() {
		log.Logf("Current svc is invalid to get associate path, %v", svcPack)
		return "", ""
	}

	var kubeconfigContent string
	var server string
	if err := Get(
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			if _, ok := dirMapping.PackToPath[svcPack.Key()]; ok {
				kubeconfigContent = dirMapping.PackToKubeConfigBytes[svcPack.Key()]
				server = dirMapping.PackToKubeConfigServer[svcPack.Key()]
			} else if _, ok := dirMapping.PackToPath[svcPack.Key()]; ok {
				kubeconfigContent = dirMapping.PackToKubeConfigBytes[svcPack.KeyWithoutContainer()]
				server = dirMapping.PackToKubeConfigServer[svcPack.KeyWithoutContainer()]
			} else if _, ok := dirMapping.PackToPath[svcPack.Key().WithoutNid()]; ok {
				kubeconfigContent = dirMapping.PackToKubeConfigBytes[svcPack.Key().WithoutNid()]
				server = dirMapping.PackToKubeConfigServer[svcPack.Key().WithoutNid()]
			} else {
				keys := svcPack.KeyWithoutContainer().WithoutNid()
				kubeconfigContent = dirMapping.PackToKubeConfigBytes[keys]
				server = dirMapping.PackToKubeConfigServer[svcPack.KeyWithoutContainer().WithoutNid()]
			}
			return nil
		},
	); err != nil {
		log.ErrorE(err, fmt.Sprintf("Current svc is fail to get associate path, %v", svcPack))
		return "", ""
	}
	return kubeconfigContent, server
}

func (svcPack *SvcPack) UnAssociatePath() {
	if !svcPack.valid() {
		log.Logf("Current svc is invalid to get associate path, %v", svcPack)
	}

	if err := Update(
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			delete(dirMapping.PackToPath, svcPack.Key())
			delete(dirMapping.PackToPath, svcPack.KeyWithoutContainer())

			delete(dirMapping.PackToPath, svcPack.Key().WithoutNid())
			delete(dirMapping.PackToPath, svcPack.KeyWithoutContainer().WithoutNid())
			return nil
		},
	); err != nil {
		log.ErrorE(err, fmt.Sprintf("Current svc is fail to get associate path, %v", svcPack))
	}
}

// return error if not found
func (d DevPath) GetDefaultPack() (*SvcPack, error) {
	return getDefaultPack(d)
}

func (d DevPath) GetAllPacks() *AllSvcPackAssociateByPath {
	return getAllPacks(d)
}

func (d DevPath) AlreadyAssociate(specifyPack *SvcPack) bool {
	for key, _ := range d.GetAllPacks().Packs {
		if key == specifyPack.Key() {
			return true
		}
	}
	return false
}

// Associate setAsDefaultSvc:
// if this dev path has been associate by svc && [setAsDefaultSvc==true]
// replace the default svc to the path
//
// setAsDefaultSvc==false when data migration
func (d DevPath) Associate(specifyPack *SvcPack, kubeconfig string, setAsDefaultSvc bool) error {
	if !specifyPack.valid() {
		return errors.New("Svc pack is invalid")
	}

	// step.1 remove all mapping from specify pack
	// step.2 build mapping from specifyPack to current path and associate kubeconfig and pack
	// step.3 mark specifyPack as default pack to current path

	return d.removePackAndThen(
		specifyPack,
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			kubeconfigContent := fp.NewFilePath(kubeconfig).ReadFile()
			server := getKubeConfigServer(kubeconfig)
			if kubeconfigContent == "" {
				log.Log("Associate Svc %s but kubeconfig is nil", specifyPack.Key())
				return nil
			}

			key := specifyPack.Key()
			keyWithoutContainer := specifyPack.KeyWithoutContainer()

			dirMapping.PackToPath[key] = d
			dirMapping.PackToKubeConfigBytes[key] = kubeconfigContent
			dirMapping.PackToKubeConfigServer[key] = server

			// if container not specified
			// cover all pack with same ns/app/type/svc
			if specifyPack.Container == "" {
				for keyItem, _ := range dirMapping.PackToPath {
					if strings.HasPrefix(string(keyItem), string(key)) {
						dirMapping.PackToPath[keyItem] = d
						dirMapping.PackToKubeConfigBytes[keyItem] = kubeconfigContent
						dirMapping.PackToKubeConfigServer[keyItem] = server
					}
				}
			} else {
				dirMapping.PackToPath[keyWithoutContainer] = d
				dirMapping.PackToKubeConfigBytes[keyWithoutContainer] = kubeconfigContent
				dirMapping.PackToKubeConfigServer[keyWithoutContainer] = server
			}

			if _, hasBeenSet := dirMapping.PathToDefaultPackKey[d]; setAsDefaultSvc || !hasBeenSet {
				dirMapping.PathToDefaultPackKey[d] = specifyPack.Key()
			}

			return nil
		},
	)
}

func getKubeConfigServer(kubeConfig string) string {
	utils, err := clientgoutils.NewClientGoUtils(kubeConfig, "")
	if err != nil {
		return ""
	}

	config, err := utils.NewFactory().ToRawKubeConfigLoader().RawConfig()
	if err != nil || config.CurrentContext == "" {
		return ""
	}

	if context, ok := config.Contexts[config.CurrentContext]; ok {
		return context.Cluster
	}

	return ""
}

func (d DevPath) RemovePack(specifyPack *SvcPack) error {
	return d.removePackAndThen(specifyPack, nil)
}

func (d DevPath) removePackAndThen(
	specifyPack *SvcPack,
	fun func(dirMapping *DevDirMapping,
		pathToPack map[DevPath][]*SvcPack) error) error {
	if !specifyPack.valid() {
		return errors.New("Svc pack is invalid")
	}

	return Update(
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			specifyPackKey := specifyPack.Key()
			devPathBefore := dirMapping.PackToPath[specifyPack.Key()]

			beforePacks := doGetAllPacks(devPathBefore, dirMapping, pathToPack)

			// step.1 remove or modify the before path's default packKey
			// step.2 remove mapping of specify pack to current path
			// stop.3 call fun

			// 1 -
			// if specify Svc has been associate with before path and if it is a default
			// pack of a path, should modify or remove the default Svc pack of the path
			//
			{
				// remove [path -> defaultSvc] directly if len==1
				if len(beforePacks.Packs) == 1 {
					delete(dirMapping.PathToDefaultPackKey, d)

					// modify [path -> defaultSvc] if defaultSvc == specifyPackKey
				} else if beforePacks.DefaultSvcPackKey == specifyPackKey {

					// modify the before path's default packKey to a random packKey
					for random, _ := range beforePacks.Packs {
						if random != specifyPackKey {
							dirMapping.PathToDefaultPackKey[devPathBefore] = random
						}
					}
				} else {
					// do not need to remove default pack key
				}
			}

			// 2 -
			delete(dirMapping.PackToPath, specifyPack.Key())

			// 3 -
			if fun == nil {
				return nil
			} else {
				return fun(dirMapping, pathToPack)
			}
		},
	)
}

func (svcPack *SvcPack) valid() bool {
	return svcPack != nil &&
		svcPack.Ns != "" && svcPack.App != "" &&
		svcPack.SvcType != "" && svcPack.Svc != ""
}

func getDefaultPack(path DevPath) (*SvcPack, error) {
	packs := getAllPacks(path)
	if packs == nil {
		return nil, NO_DEFAULT_PACK
	}
	defaultSvcPackKey := packs.DefaultSvcPackKey

	if pack, ok := packs.Packs[defaultSvcPackKey]; ok {
		return pack, nil
	}

	// some dirty data may cause defaultSvcPackKey to nil mapping
	// so return random one
	if len(packs.Packs) > 0 {
		for _, pack := range packs.Packs {
			if pack.Key() != pack.KeyWithoutContainer() {
				return pack, nil
			}
		}
	}

	return nil, NO_DEFAULT_PACK
}

// list all pack associate with this path
// this method will not access the db
func doGetAllPacks(path DevPath, dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) *AllSvcPackAssociateByPath {
	var r *AllSvcPackAssociateByPath

	packs, ok := pathToPack[path]

	defaultSvcPackKey := dirMapping.PathToDefaultPackKey[path]

	allpacks := make(map[SvcPackKey]*SvcPack, 0)
	KubeConfigs := make(map[SvcPackKey]string, 0)
	ServerMapping := make(map[SvcPackKey]string, 0)

	if ok {
		for _, pack := range packs {
			packKey := pack.Key()
			allpacks[packKey] = pack
			KubeConfigs[packKey] = dirMapping.PackToKubeConfigBytes[packKey]
			ServerMapping[packKey] = dirMapping.PackToKubeConfigServer[packKey]
		}
	}

	r = &AllSvcPackAssociateByPath{
		Packs:             allpacks,
		DefaultSvcPackKey: defaultSvcPackKey,
		Kubeconfigs:       KubeConfigs,
		ServerMapping:     ServerMapping,
	}
	return r
}

// list all pack associate with this path
func getAllPacks(path DevPath) *AllSvcPackAssociateByPath {
	var r *AllSvcPackAssociateByPath
	if err := Get(
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			r = doGetAllPacks(path, dirMapping, pathToPack)
			return nil
		},
	); err != nil {
		return nil
	}
	return r
}
