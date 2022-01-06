/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
	"path/filepath"
)

type ClusterInfo struct {
	Cluster    string
	Context    string
	NameSpace  string
	KubeConfig string // path
	User       string
	K8sVer     string `json:"-" yaml:"-"`
	NhctlVer   string `json:"-" yaml:"-"`
	k8sClient  *clientgoutils.ClientGoUtils
}

type TviewApplication struct {
	app          *tview.Application
	pages        *tview.Pages
	mainLayout   *tview.Flex
	header       *tview.Flex
	cacheView    []tview.Primitive
	maxCacheView int
	clusterInfo  *ClusterInfo
}

//type Nocalhost

func NewTviewApplication() *TviewApplication {
	t := TviewApplication{app: tview.NewApplication()}
	t.cacheView = make([]tview.Primitive, 0)
	t.maxCacheView = 10
	t.pages = tview.NewPages()
	t.mainLayout = tview.NewFlex().SetDirection(tview.FlexRow)
	mainMenu := t.buildMainMenu()
	t.header = buildHeader()
	t.clusterInfo = loadLocalClusterInfo()
	t.RefreshHeader()

	t.mainLayout.AddItem(t.header, 0, 1, false)
	t.mainLayout.AddItem(mainMenu, 0, 2, true)

	str, _ := os.Getwd()
	t.mainLayout.AddItem(
		tview.NewTextView().SetText("Current path: "+str),
		1, 1, false,
	)
	//t.mainLayout.SetBackgroundColor(tcell.ColorBlack)

	//t.pages.AddPage("Main", t.mainLayout, true, true)

	t.app.SetRoot(t.mainLayout, true).EnableMouse(true)
	t.initEventHandler()
	return &t
}

func loadLocalClusterInfo() *ClusterInfo {
	path := defaultKubeConfigPath
	if _, err := os.Stat(path); err != nil {
		path = os.Getenv("KUBECONFIG")
	}
	client, err := clientgoutils.NewClientGoUtils(path, "")
	if err != nil {
		return nil
	}
	config, err := client.ClientConfig.RawConfig()
	if err != nil {
		return nil
	}
	if len(config.Contexts) == 0 {
		return nil
	}

	currentCxt := config.Contexts[config.CurrentContext]
	if currentCxt == nil {
		return nil
	}

	k8sVer, _ := client.ClientSet.ServerVersion()
	return &ClusterInfo{
		Cluster:    currentCxt.Cluster,
		Context:    config.CurrentContext,
		NameSpace:  currentCxt.Namespace,
		User:       currentCxt.AuthInfo,
		K8sVer:     k8sVer.GitVersion,
		KubeConfig: path,
		k8sClient:  client,
	}
}

func loadAllClusterInfos() []*ClusterInfo {
	allClusterInfos := make([]*ClusterInfo, 0)

	// Load from nocalhost
	ks, err := nocalhost.GetAllKubeconfig()
	if err == nil {
		for _, k := range ks {
			allClusterInfos = append(allClusterInfos, getClusterInfoByPath(k)...)
		}
	}

	defaultPath := filepath.Join(utils.GetHomePath(), ".kube", "config")
	if _, err := os.Stat(defaultPath); err != nil {
		defaultPath = os.Getenv("KUBECONFIG")
	}

	if defaultPath != "" {
		allClusterInfos = append(allClusterInfos, getClusterInfoByPath(defaultPath)...)
	}
	return allClusterInfos
}

func getClusterInfoByPath(path string) []*ClusterInfo {
	allClusterInfos := make([]*ClusterInfo, 0)
	client, err := clientgoutils.NewClientGoUtils(path, "")
	if err != nil {
		return allClusterInfos
	}
	config, err := client.ClientConfig.RawConfig()
	if err != nil {
		return allClusterInfos
	}
	if len(config.Contexts) == 0 {
		return allClusterInfos
	}

	for s, context := range config.Contexts {
		k8sVer, _ := client.ClientSet.ServerVersion()
		allClusterInfos = append(allClusterInfos, &ClusterInfo{
			Cluster:    context.Cluster,
			Context:    s,
			NameSpace:  client.GetNameSpace(),
			KubeConfig: path,
			User:       context.AuthInfo,
			K8sVer:     k8sVer.GitVersion,
			k8sClient:  client,
		})
	}
	return allClusterInfos
}

func (t *TviewApplication) initEventHandler() {
	// Set keyboard event handler
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			t.app.Stop()
			os.Exit(0)
		case tcell.KeyEscape:
			t.switchMainMenu()
		case tcell.KeyLeft:
			t.switchBodyToPre()
		}
		return event
	})
}

func (t *TviewApplication) Run() error {
	if t.app != nil {
		return t.app.Run()
	}
	return nil
}

func (t *TviewApplication) switchBodyTo(m tview.Primitive) {
	t.mainLayout.RemoveItemAtIndex(1)
	t.mainLayout.AddItemAtIndex(1, m, 0, 2, true)
	t.app.SetFocus(m)
}

func (t *TviewApplication) switchBodyToC(from, to tview.Primitive) {
	t.cacheView = append(t.cacheView, from)
	if len(t.cacheView) > t.maxCacheView && t.maxCacheView > 1 {
		t.cacheView = t.cacheView[1:]
	}
	t.mainLayout.RemoveItemAtIndex(1)
	t.mainLayout.AddItemAtIndex(1, to, 0, 2, true)
	t.app.SetFocus(to)
}

func (t *TviewApplication) switchMainMenu() {
	t.switchBodyTo(t.buildMainMenu())
}

func (t *TviewApplication) switchBodyToPre() {
	if len(t.cacheView) > 0 {
		item := t.cacheView[len(t.cacheView)-1]
		t.cacheView = t.cacheView[0 : len(t.cacheView)-1]
		t.switchBodyTo(item)
	}
}
