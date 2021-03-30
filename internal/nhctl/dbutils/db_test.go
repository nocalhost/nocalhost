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

package dbutils

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"testing"
	"time"
)

func TestOpenLevelDB(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db", &opt.Options{
		ErrorIfMissing: true,
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("%v exists\n", db)
	}
}

func TestOpenLevelDBForPut(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db", &opt.Options{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("db opened\n")
	}

	time.Sleep(60 * time.Second)
	fmt.Println("After 60s")
	err = db.Put([]byte("aaa"), []byte("bbb"), nil)
	if err != nil {
		panic(err)
	}
}

func TestOpenLevelDBForLog(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db", &opt.Options{})
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	fmt.Printf("db opened111\n")

	fmt.Printf("in for %d", 1)
	for i := 0; i < 100; i++ {
		fmt.Println("Update ", i)
		err = db.Put([]byte("aaa"), []byte(fmt.Sprintf("bbb %d", i)), nil)
		if err != nil {
			panic(err)
		}
		time.Sleep(1 * time.Second)
	}
	defer db.Close()
}

func TestOpenLevelDBForIter(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db2", &opt.Options{ReadOnly: true})
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	defer db.Close()
	iter := db.NewIterator(nil, nil)
	if iter.Next() {
		fmt.Println(iter.Key())
	}
}

var longString = `
nh6lmaa.bookinfo.profile.v2=name: ""
releasename: ""
namespace: nh6lmaa
kubeconfig: /Users/xinxinhuang/.nh/plugin/kubeConfigs/293_config
dependencyConfigMapName: nocalhost-depends-do-not-overwrite-kyuq
appType: rawManifest
svcProfile:
- rawConfig:
    name: ratings
    serviceType: deployment
    dependLabelSelector:
      pods:
      - productpage
      - app.kubernetes.io/name=productpage
      jobs:
      - dep-job
    containers:
    - name: ""
      install: null
      dev:
        gitUrl: https://e.coding.net/codingcorp/nocalhost/bookinfo-ratings.git
        image: codingcorp-docker.pkg.coding.net/nocalhost/dev-images/node:12.18.1-slim
        shell: bash
        workDir: /home/nocalhost-dev
        resources: null
        persistentVolumeDirs: []
        command: null
        debug: null
        useDevContainer: false
        sync:
          type: send
          filePattern:
          - ./
          ignoreFilePattern:
          - .git
        env:
        - name: DEBUG
          value: "true"
        envFrom: null
        portForward: []
  containerProfile: []
  actualName: ratings
  developing: false
  portForwarded: false
  syncing: false
  remoteSyncthingPort: 0
  remoteSyncthingGUIPort: 0
  syncthingSecret: ""
  localSyncthingPort: 0
  localSyncthingGUIPort: 0
  localAbsoluteSyncDirFromDevStartPlugin: []
  devPortForwardList: []
- rawConfig:
    name: reviews
    serviceType: deployment
    dependLabelSelector:
      pods:
      - productpage
      jobs: []
    containers:
    - name: ""
      install: null
      dev:
        gitUrl: https://e.coding.net/codingcorp/nocalhost/bookinfo-reviews.git
        image: codingcorp-docker.pkg.coding.net/nocalhost/dev-images/java:latest
        shell: bash
        workDir: /home/nocalhost-dev
        resources: null
        persistentVolumeDirs: []
        command: null
        debug: null
        useDevContainer: false
        sync:
          type: send
          filePattern:
          - ./
          ignoreFilePattern:
          - .git
        env: []
        envFrom: null
        portForward: []
  containerProfile: []
  actualName: reviews
  developing: false
  portForwarded: false
  syncing: false
  remoteSyncthingPort: 0
  remoteSyncthingGUIPort: 0
  syncthingSecret: ""
  localSyncthingPort: 0
  localSyncthingGUIPort: 0
  localAbsoluteSyncDirFromDevStartPlugin: []
  devPortForwardList: []
- rawConfig:
    name: productpage
    serviceType: deployment
    dependLabelSelector:
      pods: []
      jobs:
      - dep-job
    containers:
    - name: productpage
      install:
        env: []
        envFrom:
          envFile: []
        portForward:
        - 39080:9080
      dev:
        gitUrl: https://e.coding.net/codingcorp/nocalhost/bookinfo-productpage.git
        image: codingcorp-docker.pkg.coding.net/nocalhost/dev-images/python:3.7.7-slim-productpage
        shell: bash
        workDir: /home/nocalhost-dev
        resources: null
        persistentVolumeDirs: []
        command: null
        debug: null
        useDevContainer: false
        sync:
          type: send
          filePattern:
          - ./
          ignoreFilePattern:
          - .git
        env: []
        envFrom: null
        portForward:
        - 39080:9080
  containerProfile: []
  actualName: productpage
  developing: true
  portForwarded: true
  syncing: true
  remoteSyncthingPort: 49307
  remoteSyncthingGUIPort: 49308
  syncthingSecret: productpage-nocalhost-syncthing-secret
  localSyncthingPort: 49310
  localSyncthingGUIPort: 49309
  localAbsoluteSyncDirFromDevStartPlugin:
  - /Users/xinxinhuang/nocalhost/bookinfo/bookinfo-productpage
  devPortForwardList:
  - localport: 39080
    remoteport: 9080
    role: ""
    status: LISTEN
    reason: Heart Beat
    podName: productpage-cfd7ffbf6-l5vt5
    updated: "2021-03-30 14:12:52"
    pid: 14491
    runByDaemonServer: true
    sudo: false
    daemonserverpid: 14491
  - localport: 49307
    remoteport: 49307
    role: SYNC
    status: LISTEN
    reason: Heart Beat
    podName: productpage-cfd7ffbf6-l5vt5
    updated: "2021-03-30 14:12:53"
    pid: 14491
    runByDaemonServer: true
    sudo: false
    daemonserverpid: 14491
- rawConfig:
    name: details
    serviceType: deployment
    dependLabelSelector: null
    containers:
    - name: ""
      install: null
      dev:
        gitUrl: https://e.coding.net/codingcorp/nocalhost/bookinfo-details.git
        image: codingcorp-docker.pkg.coding.net/nocalhost/dev-images/ruby:2.7.1-slim
        shell: bash
        workDir: /home/nocalhost-dev
        resources: null
        persistentVolumeDirs: []
        command: null
        debug: null
        useDevContainer: false
        sync:
          type: send
          filePattern:
          - ./
          ignoreFilePattern:
          - .git
        env:
        - name: DEBUG
          value: "true"
        envFrom: null
        portForward: []
  containerProfile: []
  actualName: details
  developing: false
  portForwarded: false
  syncing: false
  remoteSyncthingPort: 0
  remoteSyncthingGUIPort: 0
  syncthingSecret: ""
  localSyncthingPort: 0
  localSyncthingGUIPort: 0
  localAbsoluteSyncDirFromDevStartPlugin: []
  devPortForwardList: []
installed: true
syncDirs: []
resourcePath:
- manifest/templates
ignoredPath: []
onPreInstall:
- path: manifest/templates/pre-install/print-num-job-01.yaml
  weight: "1"
- path: manifest/templates/pre-install/print-num-job-02.yaml
  weight: "-5"
gitUrl: git@github.com:nocalhost/bookinfo.git
gitRef: ""
helmRepoUrl: ""
helmRepoName: ""
env:
- name: DEBUG
  value: "true"
- name: DOMAIN
  value: coding.com
envFrom:
  envFile: []


`

func TestOpenLevelDBForOpenManyTime(t *testing.T) {

	for i := 0; i < 100; i++ {
		db, err := leveldb.OpenFile("/tmp/tmp/db2", &opt.Options{ErrorIfMissing: true})
		if err != nil {
			panic(err)
		}
		fmt.Println("db opened111", i)
		err = db.Put([]byte("aaa"), []byte(fmt.Sprintf("%s %d", longString, i)), nil)
		if err != nil {
			panic(err)
		}
		time.Sleep(1 * time.Second)
		db.Close()
	}

}

func TestOpenLevelDBForCompact(t *testing.T) {

	db, err := leveldb.OpenFile("/tmp/tmp/db2", &opt.Options{ErrorIfMissing: true})
	if err != nil {
		panic(err)
	}
	// key: nh6lmaa.bookinfo.profile.v2
	s, err := db.SizeOf([]util.Range{*util.BytesPrefix([]byte("nh6lmaa.bookinfo.profile.v2"))})
	if err != nil {
		panic(err)
	}
	fmt.Println(s.Sum())

	//db.CompactRange()

	time.Sleep(1 * time.Second)
	db.Close()

}
