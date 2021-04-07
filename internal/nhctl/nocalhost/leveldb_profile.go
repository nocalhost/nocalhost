/*
Copyright 2021 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nocalhost

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/yaml.v2"
	nocalhostDb "nocalhost/internal/nhctl/nocalhost/db"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/profile"
	"os"
	"strings"
)

func UpdateProfileV2(ns, app string, profileV2 *profile.AppProfileV2) error {
	var err error
	db, err := nocalhostDb.OpenApplicationLevelDB(ns, app, false)
	if err != nil {
		return err
	}
	defer db.Close()
	bys, err := yaml.Marshal(profileV2)
	if err != nil {
		return errors.Wrap(err, "")
	}
	// Double check
	if _, err = os.Stat(nocalhost_path.GetAppDbDir(ns, app)); err != nil {
		return errors.Wrap(err, "")
	}
	return db.Put([]byte(profile.ProfileV2Key(ns, app)), bys)
}

func GetKubeConfigFromProfile(ns, app string) (string, error) {
	p, err := GetProfileV2(ns, app)
	if err != nil {
		return "", err
	}
	return p.Kubeconfig, nil
}

func GetProfileV2(ns, app string) (*profile.AppProfileV2, error) {
	var err error
	db, err := nocalhostDb.OpenApplicationLevelDB(ns, app, true)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	result := &profile.AppProfileV2{}
	bys, err := db.Get([]byte(profile.ProfileV2Key(ns, app)))
	if err != nil {
		if err == leveldb.ErrNotFound {
			result, err := db.ListAll()
			if err != nil {
				return nil, errors.Wrap(err, "")
			}
			for key, val := range result {
				if strings.Contains(key, "profile.v2") {
					bys = []byte(val)
					break
				}
			}
		} else {
			return nil, errors.Wrap(err, "")
		}
	}
	if len(bys) == 0 {
		return nil, errors.New("Profile not found")
	}

	err = yaml.Unmarshal(bys, result)
	return result, errors.Wrap(err, "")
}
