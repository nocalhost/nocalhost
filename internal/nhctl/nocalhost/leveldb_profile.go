/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package nocalhost

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/yaml.v3"
	nocalhostDb "nocalhost/internal/nhctl/nocalhost/db"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/profile"
	"os"
	"regexp"
	"strings"
)

var ProfileNotFound = errors.New("Profile Not Found")

func UpdateProfileV2(ns, app, nid string, profileV2 *profile.AppProfileV2) error {
	var err error
	db, err := nocalhostDb.OpenApplicationLevelDB(ns, app, nid, false)
	if err != nil {
		return err
	}
	defer db.Close()
	bys, err := yaml.Marshal(profileV2)
	if err != nil {
		return errors.Wrap(err, "")
	}
	// Double check
	if _, err = os.Stat(nocalhost_path.GetAppDbDir(ns, app, nid)); err != nil {
		return errors.Wrap(err, "")
	}
	profileV2.GenerateIdentifierIfNeeded()
	return db.Put([]byte(profile.ProfileV2Key(ns, app)), bys)
}

func GetKubeConfigFromProfile(ns, app, nid string) (string, error) {
	p, err := GetProfileV2(ns, app, nid)
	if err != nil {
		return "", err
	}
	return p.Kubeconfig, nil
}

func GetProfileV2(ns, app, nid string) (*profile.AppProfileV2, error) {
	var err error
	db, err := nocalhostDb.OpenApplicationLevelDB(ns, app, nid, true)
	if err != nil {
		return nil, err
	}
	defer db.Close()

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
		return nil, errors.Wrap(ProfileNotFound, "")
	}

	result, err := UnmarshalProfileUnStrict(bys)
	return result, errors.Wrap(err, "")
}

func UnmarshalProfileUnStrict(p []byte) (*profile.AppProfileV2, error) {
	result := &profile.AppProfileV2{}
	err := yaml.Unmarshal(p, result)
	if err != nil {
		re, _ := regexp.Compile("remoteDebugPort: \"[0-9]*\"") // fix string convert int error
		rep := re.ReplaceAllString(string(p), "")
		err = yaml.Unmarshal([]byte(rep), result)
	}
	return result, err
}
