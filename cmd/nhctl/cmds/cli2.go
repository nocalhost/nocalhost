///*
//* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
//* This source code is licensed under the Apache License Version 2.0.
// */
//
package cmds

//
//import (
//	surveypkg "github.com/AlecAivazis/survey/v2"
//	"github.com/pkg/errors"
//	"github.com/spf13/cobra"
//	"io/ioutil"
//	"nocalhost/internal/nhctl/appmeta"
//	"nocalhost/pkg/nhctl/log"
//	yaml "nocalhost/pkg/nhctl/utils/custom_yaml_v3"
//	"os"
//	"regexp"
//	"sort"
//	"strings"
//)
//
//func init() {
//	rootCmd.AddCommand(cliCommand)
//}
//
//var cliCommand = &cobra.Command{
//	Use:   "cli",
//	Short: "",
//	Long:  ``,
//	Run: func(cmd *cobra.Command, args []string) {
//
//		deployANewOne := "deploy a new application"
//		useExistOne := "use a existing one"
//
//		q := &QuestionOptions{
//			Question:     "Which application you want to use?",
//			DefaultValue: deployANewOne,
//			Options: []string{
//				deployANewOne,
//				useExistOne,
//			},
//		}
//
//		a, err := Question(q)
//		if err != nil {
//			log.FatalE(err, "")
//		}
//
//		//answerKeyValue := make(map[string]string, 0)
//		devConfig := &DevConfig{}
//		//applicationName := ""
//		if a == deployANewOne {
//			quickStartApp := "quickstart: Use demo application bookinfo"
//			helmApp := "helm: Use my own Helm chart (e.g. local via ./chart/ or any remote chart)"
//			kubectlApp := "kubectl: Use existing Kubernetes manifests (e.g. ./kube/deployment.yaml)"
//			kustomizeApp := "kustomize: Use an existing Kustomization (e.g. ./kube/kustomization/)"
//			q = &QuestionOptions{
//				Question:     "Which type of application you want to deploy?",
//				DefaultValue: quickStartApp,
//				Options: []string{
//					quickStartApp,
//					helmApp,
//					kubectlApp,
//					kustomizeApp,
//				},
//			}
//
//			a, err := Question(q)
//			if err != nil {
//				log.FatalE(err, "")
//			}
//
//			if a == quickStartApp {
//				must(Prepare())
//				installFlags.AppType = string(appmeta.ManifestGit)
//				installFlags.GitUrl = "https://github.com/nocalhost/bookinfo.git"
//				installApplication("bookinfo")
//				devConfig.Application = "bookinfo"
//			}
//		} else {
//			must(Prepare())
//			appList := ListApplicationsResult()
//			appOptions := make([]string, 0)
//			for _, n := range appList {
//				for _, applicationInfo := range n.Application {
//					appOptions = append(appOptions, applicationInfo.Name)
//				}
//			}
//			q = &QuestionOptions{
//				Question:     "Which application you want to use?",
//				DefaultValue: appOptions[0],
//				Options:      appOptions,
//			}
//			if devConfig.Application, err = Question(q); err != nil {
//				log.FatalE(err, "")
//			}
//		}
//
//		r := get("all", "", false)
//		if r == nil {
//			log.Fatal("No workload found")
//		}
//		workloadList := make([]string, 0)
//		for _, result := range r {
//			for _, app := range result.Application {
//				if app.Name != devConfig.Application {
//					continue
//				}
//				for _, group := range app.Groups {
//					if group.GroupName == "Workloads" {
//						for _, resource := range group.List {
//							for _, item := range resource.List {
//								if _, n, err := getNamespaceAndName(item.Metadata); err != nil {
//									continue
//								} else {
//									workloadList = append(workloadList, resource.Name+"/"+n)
//								}
//							}
//						}
//					}
//				}
//			}
//		}
//		q = &QuestionOptions{
//			Question:     "Which workload you want to develop?",
//			DefaultValue: workloadList[0],
//			Options:      workloadList,
//		}
//		if wl, err := Question(q); err == nil {
//			strs := strings.Split(wl, "/")
//			devConfig.Type = strings.TrimSuffix(strs[0], "s")
//			devConfig.WorkLoad = strs[1]
//		} else {
//			log.FatalE(err, "")
//		}
//
//		initAppAndCheckIfSvcExist(devConfig.Application, devConfig.WorkLoad, devConfig.Type)
//		//client, err := clientgoutils.NewClientGoUtils(kubeConfig, nameSpace)
//		//if err != nil {
//		//	log.FatalE(err, "")
//		//}
//		//containers, err := controller.GetOriginalContainers(devConfig.WorkLoad, base.SvcType(devConfig.Type), client)
//		containers, err := nocalhostSvc.GetOriginalContainers()
//		if err != nil {
//			log.FatalE(err, "")
//		}
//		containerList := make([]string, 0)
//		for _, c := range containers {
//			containerList = append(containerList, c.Name)
//		}
//		q = &QuestionOptions{
//			Question:     "Which container you want to develop?",
//			DefaultValue: containerList[0],
//			Options:      containerList,
//		}
//		if devConfig.Container, err = Question(q); err != nil {
//			log.FatalE(err, "")
//		}
//
//		image := getConfig(devConfig.Container, "image")
//		if image == "" {
//			devImages := []string{
//				"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/java:11",
//				"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/ruby:3.0",
//				"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/node:14",
//				"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/python:3.9",
//				"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/golang:1.16",
//				"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/perl:latest",
//				"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/rust:latest",
//				"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/php:latest",
//			}
//			q = &QuestionOptions{
//				Question:     "Which dev image you want to use?",
//				DefaultValue: devImages[0],
//				Options:      devImages,
//			}
//			if devImage, err := Question(q); err != nil {
//				log.FatalE(err, "")
//			} else {
//				setConfig(devConfig.Container, "image", devImage)
//			}
//		}
//
//		devConfigBytes, _ := yaml.Marshal(devConfig)
//		if err = ioutil.WriteFile("nocalhost.yaml", devConfigBytes, 0755); err != nil {
//			log.FatalE(err, "")
//		}
//
//		q = &QuestionOptions{
//			Question:     "Do you want to start develop now?",
//			DefaultValue: "no",
//			Options:      []string{"yes", "no"},
//		}
//		if a, err = Question(q); a == "yes" {
//			wd, _ := os.Getwd()
//			devStartOps.LocalSyncDir = []string{wd}
//			devStartOps.Container = devConfig.Container
//			startDev()
//		}
//	},
//}
//
//type DevConfig struct {
//	Application string
//	WorkLoad    string
//	Container   string
//	Type        string
//}
//
//type QuestionOptions struct {
//	Question               string
//	DefaultValue           string
//	ValidationRegexPattern string
//	ValidationMessage      string
//	ValidationFunc         func(value string) error
//	Options                []string
//	Sort                   bool
//	IsPassword             bool
//}
//
//var DefaultValidationRegexPattern = regexp.MustCompile("^.*$")
//
//func Question(params *QuestionOptions) (string, error) {
//	var prompt surveypkg.Prompt
//	compiledRegex := DefaultValidationRegexPattern
//	if params.ValidationRegexPattern != "" {
//		compiledRegex = regexp.MustCompile(params.ValidationRegexPattern)
//	}
//
//	if params.Options != nil {
//		if params.Sort {
//			params.Options = copyStringArray(params.Options)
//			sort.Strings(params.Options)
//		}
//
//		prompt = &surveypkg.Select{
//			Message: params.Question,
//			Options: params.Options,
//			Default: params.DefaultValue,
//		}
//	} else if params.IsPassword {
//		prompt = &surveypkg.Password{
//			Message: params.Question,
//		}
//	} else {
//		prompt = &surveypkg.Input{
//			Message: params.Question,
//			Default: params.DefaultValue,
//		}
//	}
//
//	question := []*surveypkg.Question{
//		{
//			Name:   "question",
//			Prompt: prompt,
//		},
//	}
//
//	if params.Options == nil {
//		question[0].Validate = func(val interface{}) error {
//			str, ok := val.(string)
//			if !ok {
//				return errors.New("Input was not a string")
//			}
//
//			// Check regex
//			if !compiledRegex.MatchString(str) {
//				if params.ValidationMessage != "" {
//					return errors.New(params.ValidationMessage)
//				}
//
//				return errors.Errorf("Answer has to match pattern: %s", compiledRegex.String())
//			}
//
//			// Check function
//			if params.ValidationFunc != nil {
//				err := params.ValidationFunc(str)
//				if err != nil {
//					if params.ValidationMessage != "" {
//						return errors.New(params.ValidationMessage)
//					}
//
//					return errors.Errorf("%v", err)
//				}
//			}
//
//			return nil
//		}
//	}
//
//	// Ask it
//	answers := struct {
//		Question string
//	}{}
//
//	err := surveypkg.Ask(question, &answers)
//	if err != nil {
//		// Keyboard interrupt
//		os.Exit(0)
//	}
//	if answers.Question == "" && len(params.Options) > 0 {
//		answers.Question = params.Options[0]
//	}
//
//	return answers.Question, nil
//}
//
//func copyStringArray(strings []string) []string {
//	retStrings := []string{}
//	retStrings = append(retStrings, strings...)
//	return retStrings
//}
