/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmds

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"nocalhost/cmd/nhctl/cmds/tpl"
)

type CommonFlags struct {
	AppName   string
	SvcName   string
	AppConfig bool
}

var commonFlags = CommonFlags{}

func init() {
	configTemplateCmd.Flags().StringVarP(&commonFlags.SvcName, "deployment", "d", "", "k8s deployment which your developing service exists")
	configCmd.AddCommand(configTemplateCmd)
}

var configTemplateCmd = &cobra.Command{
	Use:   "template [Name]",
	Short: "get service config template",
	Long:  "get service config template",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		commonFlags.AppName = args[0]
		initAppAndCheckIfSvcExist(commonFlags.AppName, commonFlags.SvcName, nil)
		t, err := tpl.GetSvcTpl(commonFlags.SvcName)
		mustI(err, "fail to get svc tpl")
		fmt.Println(t)
	},
}
