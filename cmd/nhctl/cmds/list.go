/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	common2 "nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/common"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/daemon_handler"
	"nocalhost/internal/nhctl/model"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
)

var listFlags = &app_flags.ListFlags{}

func init() {
	listCmd.Flags().BoolVar(&listFlags.Yaml, "yaml", false, "use yaml as out put, only supports for 'nhctl list'")
	listCmd.Flags().BoolVar(&listFlags.Json, "json", false, "use json as out put, only supports for 'nhctl list'")
	listCmd.Flags().BoolVar(&listFlags.Full, "full", false, "list application meta full")
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:     "list [NAME]",
	Aliases: []string{"ls"},
	Short:   "List applications",
	Long:    `List applications`,
	Run: func(cmd *cobra.Command, args []string) {
		must(common2.Prepare())

		if len(args) > 0 { // list application detail
			nocalhostApp, err := common2.InitApp(args[0])
			must(err)
			ListApplicationSvc(nocalhostApp)
			os.Exit(0)
		}

		if listFlags.Yaml {
			ListApplicationsYaml()
		} else if listFlags.Json {
			ListApplicationsJson()
		} else if listFlags.Full {
			ListApplicationsFull()
		} else {
			ListApplications()
		}
	},
}

func ListApplicationSvc(napp *app.Application) {
	var data [][]string
	appProfile, _ := napp.GetProfile()
	for _, svcProfile := range appProfile.SvcProfile {
		rols := []string{
			svcProfile.GetName(), strconv.FormatBool(svcProfile.Developing), strconv.FormatBool(svcProfile.Syncing),
			fmt.Sprintf("%v", svcProfile.DevPortForwardList),
			fmt.Sprintf("%s", svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin),
			strconv.Itoa(svcProfile.LocalSyncthingGUIPort),
		}
		data = append(data, rols)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "DEVELOPING", "SYNCING", "DEV-PORT-FORWARDED", "SYNC-PATH", "LOCAL-SYNCTHING-GUI"})

	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output
}

func ListApplicationsResult() []*model.Namespace {
	metas, err := DoGetApplicationMetas()
	must(err)
	return daemon_handler.ParseApplicationsResult(common2.NameSpace, metas)
}

func ListApplicationsFull() {
	metas, err := DoGetApplicationMetas()
	must(err)
	marshal, _ := yaml.Marshal(metas.Desc())
	fmt.Print(string(marshal))
}

func ListApplicationsJson() {
	result := ListApplicationsResult()
	marshal, _ := json.Marshal(result)
	fmt.Print(string(marshal))
}

func ListApplicationsYaml() {
	result := ListApplicationsResult()
	marshal, _ := yaml.Marshal(result)
	fmt.Print(string(marshal))
}

func ListApplications() {
	metas, err := DoGetApplicationMetas()
	must(err)
	fmt.Printf("%-20s %-20s %-20s %-20s\n", "NAME", "STATE", "NAMESPACE", "TYPE")
	for _, meta := range metas {
		fmt.Printf("%-20s %-20s %-20s %-20s\n", meta.Application, meta.ApplicationState, meta.Ns, meta.ApplicationType)
	}
}

// do get application metas
// and create default application if needed
func DoGetApplicationMetas() (appmeta.ApplicationMetas, error) {
	metas, err := nocalhost.GetApplicationMetas(common2.NameSpace, common2.KubeConfig)

	if metas == nil {
		metas = make(appmeta.ApplicationMetas, 0)
	}

	var foundDefaultApp bool
	for _, meta := range metas {
		if meta.Application == _const.DefaultNocalhostApplication && meta.IsInstalled() {
			foundDefaultApp = true
			break
		}
	}

	if !foundDefaultApp {
		// try init default application
		nocalhostApp, err := common.InitDefaultApplicationInCurrentNs(
			_const.DefaultNocalhostApplication, common2.NameSpace, common2.KubeConfig,
		)

		// if current user has not permission to create secret,
		// we also create a fake 'default.application'
		// app meta for him
		if err != nil {
			log.Logf("failed to init default application in namespace: %s", common2.NameSpace)
			metas = append(metas, appmeta.FakeAppMeta(common2.NameSpace, _const.DefaultNocalhostApplication))
		} else {
			metas = append(metas, nocalhostApp.GetAppMeta())
		}
	}

	return metas, err
}
