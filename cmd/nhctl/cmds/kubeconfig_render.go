/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"fmt"
	yaml2 "github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/kubectl/pkg/cmd/portforward"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strconv"
	"strings"
	"time"
)

var opts = &portforward.PortForwardOptions{PortForwarder: &clientgoutils.ClientgoPortForwarder{}}
var factory cmdutil.Factory
var kubeConfigFlags *genericclioptions.ConfigFlags
var tempFile string

func init() {
	kubeconfigCmd.AddCommand(kubeconfigRenderCmd)
	flags := kubeconfigRenderCmd.PersistentFlags()
	kubeConfigFlags = genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeconfigRenderCmd.Flags().StringSliceVar(&opts.Address, "address", []string{"localhost"},
		"Addresses to listen on (comma separated). Only accepts IP addresses or localhost as a value. When localhost is supplied, kubectl will try to bind on both 127.0.0.1 and ::1 and will fail if neither of these addresses are available to bind.")
	cmdutil.AddPodRunningTimeoutFlag(kubeconfigRenderCmd, 60*time.Second)
	kubeConfigFlags.AddFlags(flags)
	matchVersionFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionFlags.AddFlags(flags)
	factory = cmdutil.NewFactory(matchVersionFlags)
}

var kubeconfigRenderCmd = &cobra.Command{
	Use:   "render",
	Short: "render kubeconfig",
	Long:  "port-forward TYPE/NAME [options] [LOCAL_PORT:]REMOTE_PORT [...[LOCAL_PORT_N:]REMOTE_PORT_N]",
	Run: func(cmd *cobra.Command, args []string) {
		if *kubeConfigFlags.KubeConfig == "-" {
			// TODO: Consider adding a flag to force to UTF16, apparently some
			// Windows tools don't write the BOM
			utf16bom := unicode.BOMOverride(unicode.UTF8.NewDecoder())
			reader := transform.NewReader(os.Stdin, utf16bom)
			content, _ := ioutil.ReadAll(reader)
			temp, _ := os.CreateTemp("", "")
			tempFile = temp.Name()
			_, _ = temp.Write(content)
			_ = temp.Chmod(0600)
			_ = temp.Close()
			kubeConfigFlags.KubeConfig = &tempFile
		}

		cmdutil.CheckErr(opts.Complete(factory, cmd, args))
		cmdutil.CheckErr(opts.Validate())
		go func() {
			cmdutil.CheckErr(opts.RunPortForward())
		}()
		<-opts.ReadyChannel
		if f, o := opts.PortForwarder.(interface {
			GetPorts() ([]clientgoutils.ForwardedPort, error)
		}); o {
			ports, err := f.GetPorts()
			if err != nil {
				log.Fatal(err)
			}
			config, err := factory.ToRawKubeConfigLoader().RawConfig()
			if context, ok := config.Contexts[config.CurrentContext]; ok {
				if cluster, ok := config.Clusters[context.Cluster]; ok {
					cluster.Server =
						cluster.Server[:strings.LastIndex(cluster.Server, ":")+1] +
							strconv.Itoa(int(ports[0].Local))
				}
			}

			kubeStruct := &clientcmdapiv1.Config{Kind: "Config", APIVersion: "v1"}
			err = clientcmdapiv1.Convert_api_Config_To_v1_Config(&config, kubeStruct, nil)

			jsonBytes, err := json.Marshal(kubeStruct)
			must(err)
			kubeYaml, err := yaml2.JSONToYAML(jsonBytes)
			must(err)
			fmt.Println(string(kubeYaml))
			fmt.Println(io.EOF)
			<-opts.StopChannel
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		_ = os.Remove(tempFile)
	},
}
