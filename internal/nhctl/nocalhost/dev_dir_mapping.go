package nocalhost

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/pkg/nhctl/log"
)

// get associate path of svcPack
// if no path match, try with svc with none container
func (svcPack *SvcPack) GetAssociatePath() DevPath {
	if !svcPack.valid() {
		log.Infof("Current svc is invalid to get associate path, %v", svcPack)
		return ""
	}

	var path DevPath
	if err := Get(
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			if _, ok := dirMapping.PackToPath[svcPack.key()]; ok {
				path = dirMapping.PackToPath[svcPack.key()]
			} else {
				path = dirMapping.PackToPath[svcPack.keyWithoutContainer()]
			}
			return nil
		},
	); err != nil {
		log.ErrorE(err, fmt.Sprintf("Current svc is fail to get associate path, %v", svcPack))
		return ""
	}
	return path
}

func (svcPack *SvcPack) UnAssociatePath()  {
	if !svcPack.valid() {
		log.Logf("Current svc is invalid to get associate path, %v", svcPack)
	}

	if err := Update(
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			delete(dirMapping.PackToPath,svcPack.key())
			delete(dirMapping.PackToPath,svcPack.keyWithoutContainer())
			return nil
		},
	); err != nil {
		log.ErrorE(err, fmt.Sprintf("Current svc is fail to get associate path, %v", svcPack))
	}
}

func (d DevPath) GetDefaultPack() (*SvcPack, error) {
	return getDefaultPack(d)
}

func (d DevPath) GetAllPacks() *AllSvcPackAssociateByPath {
	return getAllPacks(d)
}

func (d DevPath) Associate(specifyPack *SvcPack) error {
	if !specifyPack.valid() {
		return errors.New("Svc pack is invalid")
	}

	// step.1 remove all mapping from specify pack
	// step.2 build mapping from specifyPack to current path
	// step.3 mark specifyPack as default pack to current path

	return d.removePackAndThen(
		specifyPack,
		func(dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) error {
			dirMapping.PackToPath[specifyPack.key()] = d
			dirMapping.PathToDefaultPackKey[d] = specifyPack.key()
			return nil
		},
	)
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
			specifyPackKey := specifyPack.key()
			devPathBefore := dirMapping.PackToPath[specifyPack.key()]

			beforePacks := doGetAllPacks(devPathBefore, dirMapping, pathToPack)

			// step.1 remove or modify the before path's default packKey
			// step.2 remove mapping of specify pack to current path
			// stop.3 call fun

			// 1 -
			// if specify Svc has been associate with before path and if it is a default
			// pack of a path, should modify or remove the default Svc pack of the path
			//
			if beforePacks.defaultSvcPackKey == specifyPackKey {

				if len(beforePacks.packs) == 1 {
					delete(dirMapping.PathToDefaultPackKey, d)
				} else {

					// modify the before path's default packKey to a random packKey
					for packKey, _ := range beforePacks.packs {
						if packKey != specifyPackKey {
							dirMapping.PathToDefaultPackKey[d] = packKey
						}
					}
				}
			}

			// 2 -
			delete(dirMapping.PackToPath, specifyPack.key())

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
		svcPack.ns != "" && svcPack.app != "" &&
		svcPack.svcType != "" && svcPack.svc != ""
}

func getDefaultPack(path DevPath) (*SvcPack, error) {
	packs := getAllPacks(path)
	defaultSvcPackKey := packs.defaultSvcPackKey

	if pack, ok := packs.packs[defaultSvcPackKey]; ok {
		return pack, nil
	}

	return nil, errors.New("Current Svc pack not found ")
}

// list all pack associate with this path
// this method will not access the db
func doGetAllPacks(path DevPath, dirMapping *DevDirMapping, pathToPack map[DevPath][]*SvcPack) *AllSvcPackAssociateByPath {
	var r *AllSvcPackAssociateByPath

	packs, ok := pathToPack[path]
	defaultSvcPackKey := dirMapping.PathToDefaultPackKey[path]

	result := make(map[SvcPackKey]*SvcPack, 0)
	if ok {
		for _, pack := range packs {
			result[pack.key()] = pack
		}
	}

	r = &AllSvcPackAssociateByPath{
		packs:             result,
		defaultSvcPackKey: defaultSvcPackKey,
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
