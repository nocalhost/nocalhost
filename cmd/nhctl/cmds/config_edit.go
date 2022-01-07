/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */
package cmds

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/config_validate"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"os"
	"strings"

	"nocalhost/pkg/nhctl/log"
)

type ConfigEditFlags struct {
	CommonFlags
	Content   string
	file      string
	AppConfig bool
}

var configEditFlags = ConfigEditFlags{}

func init() {
	configEditCmd.Flags().StringVarP(
		&configEditFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	configEditCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	configEditCmd.Flags().StringVarP(
		&configEditFlags.Content, "content", "c", "",
		"base64 encode json content",
	)
	configEditCmd.Flags().StringVarP(
		&configEditFlags.file, "filename", "f", "",
		"that contains the configuration to edit, you can use '-f -' to pass stdin",
	)
	configEditCmd.Flags().BoolVar(&configEditFlags.AppConfig, "app-config", false, "edit application config")
	configCmd.AddCommand(configEditCmd)
}

var configEditCmd = &cobra.Command{
	Use:   "edit [Name]",
	Short: "edit service config",
	Long:  "edit service config",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		configEditFlags.AppName = args[0]

		common.InitApp(configEditFlags.AppName)

		if len(configEditFlags.Content) == 0 && len(configEditFlags.file) == 0 {
			log.Fatal("one of --content or --filename is required")
		}

		var unmashaler func(interface{}) error

		// first, resolve config from content or file
		if len(configEditFlags.Content) > 0 {

			content, err := base64.StdEncoding.DecodeString(configEditFlags.Content)
			mustI(err, "--content must be a valid base64 string")

			unmashaler = func(i interface{}) error {
				return json.Unmarshal(content, i)
			}

		} else if configEditFlags.file == "-" { // from sdtin

			// TODO: Consider adding a flag to force to UTF16, apparently some
			// Windows tools don't write the BOM
			utf16bom := unicode.BOMOverride(unicode.UTF8.NewDecoder())
			reader := transform.NewReader(os.Stdin, utf16bom)

			content, err := ioutil.ReadAll(reader)
			must(err)

			unmashaler = func(i interface{}) error {
				return yaml.Unmarshal(content, i)
			}
		} else {
			text, err := fp.NewFilePath(configEditFlags.file).ReadFileCompel()
			must(err)

			unmashaler = func(i interface{}) error {
				return yaml.Unmarshal([]byte(text), i)
			}
		}

		// set application config, plugin do not provide services struct, update application config only
		if configEditFlags.AppConfig {
			applicationConfig := &profile.ApplicationConfig{}
			must(errors.Wrap(unmashaler(applicationConfig), "fail to unmarshal content"))
			must(common.NocalhostApp.SaveAppProfileV2(applicationConfig))
			return
		}

		svcConfig := &profile.ServiceConfigV2{}
		common.CheckIfSvcExist(configEditFlags.SvcName, common.ServiceType)

		if err := unmashaler(svcConfig); err != nil {
			log.Fatal(err)
		}

		containers, _ := common.NocalhostSvc.GetOriginalContainers()
		config_validate.PrepareForConfigurationValidate(common.NocalhostApp.GetClient(), containers)
		if err := config_validate.Validate(svcConfig); err != nil {
			log.Fatal(err)
		}

		ot := svcConfig.Type
		svcConfig.Type = strings.ToLower(svcConfig.Type)
		if !nocalhost.CheckIfResourceTypeIsSupported(base.SvcType(svcConfig.Type)) {
			must(errors.New(fmt.Sprintf("Service Type %s is unsupported", ot)))
		}
		must(common.NocalhostSvc.UpdateConfig(*svcConfig))
	},
}
