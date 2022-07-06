/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"fmt"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"go.uber.org/zap/zapcore"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
	body         *tview.Flex
	treeInBody   *tview.TreeView
	rightInBody  tview.Primitive
	bottom       *tview.TextView
	cacheView    []tview.Primitive
	maxCacheView int
	clusterInfo  *ClusterInfo
}

const (
	mainPage      = "MainPage"
	devImagesPage = "DevImagesPage"
)

func NewTviewApplication() *TviewApplication {
	os.Setenv("LC_CTYPE", "en_US.UTF-8")
	t := TviewApplication{app: tview.NewApplication()}
	t.cacheView = make([]tview.Primitive, 0)
	t.maxCacheView = 10
	t.pages = tview.NewPages()
	t.mainLayout = tview.NewFlex().SetDirection(tview.FlexRow)
	t.header = buildHeader()
	t.clusterInfo = loadLocalClusterInfo()
	if t.clusterInfo == nil {
		fmt.Println(`Failed to get cluster info, please make sure your ~/.kube/config or $KUBECONFIG is configured correctly`)
		return nil
	}
	t.RefreshHeader()
	t.mainLayout.AddItem(t.header, 6, 1, false)
	t.buildTreeBody()

	t.mainLayout.SetBackgroundColor(backgroundColor)

	str, _ := os.Getwd()
	foot := tview.NewTextView().SetText("Current path: " + str)
	foot.SetBackgroundColor(backgroundColor)
	t.mainLayout.AddItem(foot, 1, 1, false)

	bottom := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetTextColor(tcell.ColorPurple)
	bottom.SetBackgroundColor(backgroundColor)
	t.mainLayout.AddItem(bottom, 1, 1, false)
	t.bottom = bottom

	t.pages.AddPage(mainPage, t.mainLayout, true, true)
	t.app.SetRoot(t.pages, true).EnableMouse(true)
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

	k8sV := "NA"
	k8sVer, err := client.ClientSet.ServerVersion()
	if err == nil {
		k8sV = k8sVer.GitVersion
	}
	return &ClusterInfo{
		Cluster:    currentCxt.Cluster,
		Context:    config.CurrentContext,
		NameSpace:  currentCxt.Namespace,
		User:       currentCxt.AuthInfo,
		K8sVer:     k8sV,
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
		ci := &ClusterInfo{
			Cluster:    context.Cluster,
			Context:    s,
			NameSpace:  client.GetNameSpace(),
			KubeConfig: path,
			User:       context.AuthInfo,
			k8sClient:  client,
		}
		if k8sVer != nil {
			ci.K8sVer = k8sVer.GitVersion
		}
		allClusterInfos = append(allClusterInfos, ci)
	}
	return allClusterInfos
}

func (t *TviewApplication) initEventHandler() {
	// Set keyboard event handler
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			t.app.Stop()
			updateLastPosition(lastPosition)
			os.Exit(0)
		case tcell.KeyCtrlL:
			t.showColorPage()
		case tcell.KeyCtrlS:
			t.buildSelectContextMenu()
		}
		return event
	})
}

var colorPageShow = false

func (t *TviewApplication) showColorPage() {
	colorPageName := "Color"
	if !t.pages.HasPage(colorPageName) {
		colorList := make([]tcell.Color, 0)

		var ii uint64 = 0
		for ; ii < 400; ii++ {
			colorList = append(colorList, tcell.Color(uint64(tcell.ColorValid)+ii))
		}

		table := NewSelectableTable("Color")
		table.SetRect(10, 10, 120, 45)
		i := 0
		for _, color := range colorList {
			table.SetCell(i/10, i%10, tview.NewTableCell(strconv.Itoa(int(color))).SetTextColor(color))
			i++
		}
		table.SetSelectedFunc(func(row, column int) {
			colorInt, _ := strconv.Atoi(table.GetCell(row, column).Text)
			t.pages.HidePage(colorPageName)
			colorPageShow = false
			go func() {
				t.app.QueueUpdateDraw(func() {
					t.app.SetFocus(t.treeInBody)
					t.UpdateTreeColor(tcell.Color(colorInt))
				})
			}()
		})
		t.pages.AddPage(colorPageName, table, false, true)
	}
	if !colorPageShow {
		t.pages.ShowPage(colorPageName)
	} else {
		t.pages.HidePage(colorPageName)
	}
	colorPageShow = !colorPageShow
}

func (t *TviewApplication) Run() error {
	if t.app != nil {
		return t.app.Run()
	}
	return nil
}

func (t *TviewApplication) switchRightBodyTo(m tview.Primitive) {
	t.body.RemoveItemAtIndex(1)
	t.body.AddItemAtIndex(1, m, 0, rightBodyProportion, true)
	t.app.SetFocus(m)
}

func (t *TviewApplication) switchRightBodyToC(from, to tview.Primitive) {
	if from != nil {
		t.cacheView = append(t.cacheView, from)
		if len(t.cacheView) > t.maxCacheView && t.maxCacheView > 1 {
			t.cacheView = t.cacheView[1:]
		}
	}
	t.body.RemoveItemAtIndex(1)
	t.body.AddItemAtIndex(1, to, 0, rightBodyProportion, true)
	t.app.SetFocus(to)
}

// Using WriteSyncer write text to TextView
func (t *TviewApplication) switchBodyToScrollingView(title string, from tview.Primitive) zapcore.WriteSyncer {
	to := t.NewScrollingTextViewForBody(title)
	to.SetBorderPadding(0, 0, 1, 0)
	sbd := SyncBuilder{func(p []byte) (int, error) {
		t.app.QueueUpdateDraw(func() {
			to.Write(p)
		})
		return 0, nil
	}}

	t.switchRightBodyToC(from, to)
	return &sbd
}

func (t *TviewApplication) switchBodyToPre() {
	if len(t.cacheView) > 0 {
		item := t.cacheView[len(t.cacheView)-1]
		t.cacheView = t.cacheView[0 : len(t.cacheView)-1]
		t.switchRightBodyTo(item)
	}
}

func (t *TviewApplication) showErr(err error, okFunc func()) {
	if err != nil {
		return
	}
	modal := tview.NewModal().
		SetText(err.Error()).
		AddButtons([]string{"Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Ok" {
				t.pages.HidePage("ErrPage")
				if okFunc != nil {
					okFunc()
				}
			}
		})
	modal.SetRect(30, 30, 30, 30)
	t.pages.AddPage("ErrPage", modal, false, true)
	t.pages.ShowPage("ErrPage")
	t.app.SetFocus(modal)
}

func (t *TviewApplication) ConfirmationBox(str string, ok, cancel func()) {
	pageName := "ConfirmationBoxPage"
	modal := tview.NewModal().
		SetText(str).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				t.pages.HidePage(pageName)
				if ok != nil {
					ok()
				}
			} else if buttonLabel == "No" {
				t.pages.HidePage(pageName)
				if cancel != nil {
					cancel()
				}
			}
		})
	_, _, w, h := t.mainLayout.GetRect()
	modal.SetRect(w/2-15, h/2-30, 30, 30)
	modal.SetBackgroundColor(tcell.ColorNavy)
	t.pages.AddPage(pageName, modal, false, true)
	t.pages.ShowPage(pageName)
	t.app.SetFocus(modal)
}

func (t *TviewApplication) InfoBox(str string, ok func()) {
	pageName := "InfoBoxPage"
	modal := tview.NewModal().
		SetText(str).
		AddButtons([]string{"Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Ok" {
				t.pages.HidePage(pageName)
				if ok != nil {
					ok()
				}
			}
		})
	_, _, w, h := t.mainLayout.GetRect()
	modal.SetRect(w/2-15, h/2-30, 30, 30)
	modal.SetBackgroundColor(tcell.ColorNavy)
	t.pages.AddPage(pageName, modal, false, true)
	t.pages.ShowPage(pageName)
	t.app.SetFocus(modal)
}

func (t *TviewApplication) NewBorderedTable(s string) *EnhancedTable {
	tab := NewRowSelectableTable(s)
	tab.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			t.switchBodyToPre()
		}
		return event
	})
	return tab
}

func (t *TviewApplication) NewScrollingTextViewForBody(title string) *tview.TextView {
	tex := NewScrollingTextView(title)
	tex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			t.switchBodyToPre()
		}
		return event
	})
	return tex
}

func (t *TviewApplication) QueueUpdateDraw(f func()) {
	if t.app != nil {
		go func() {
			t.app.QueueUpdateDraw(f)
		}()
	}
}

var lastUpdate time.Time

func (t *TviewApplication) ShowInfo(str string) {
	t.bottom.SetText(str)
	lastUpdate = time.Now()
	go func(startTime time.Time) {
		time.Sleep(3 * time.Second)
		if !startTime.Equal(lastUpdate) {
			return
		}
		t.QueueUpdateDraw(func() {
			t.bottom.SetText("")
		})
	}(lastUpdate)
}
