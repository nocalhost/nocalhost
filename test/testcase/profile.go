/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package testcase

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/test/runner"
	"strings"
	"time"
)

type containerImage struct {
	Image string `json:"image"`
}

func ProfileGetUbuntuWithJson(nhctl runner.Client) error {
	return profileGetWithJson(nhctl, "ubuntu", "any")
}

func ProfileGetDetailsWithoutJson(nhctl runner.Client) error {
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
	cmd := nhctl.GetNhctl().Command(
		ctx, "profile",
		"get", "bookinfo",
		"-d", "details", "-t", "deployment", "--container", "details", "--key", "image",
	)
	stdout, stderr, err := runner.Runner.Run(nhctl.SuiteName(), cmd)
	if err != nil {
		return err
	}
	if stderr != "" {
		return errors.New(stderr)
	}
	if stdout != "" {
		return errors.New(fmt.Sprintf("output: %s, profile get should be nil", stdout))
	}
	return nil
}

func ProfileSetDetails(nhctl runner.Client) error {
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
	cmd := nhctl.GetNhctl().Command(
		ctx, "profile",
		"set", "bookinfo",
		"-d", "details", "-t", "deployment", "--container", "details", "--key", "image",
		"--value", "helloWorld",
	)
	err := runner.Runner.RunWithCheckResult(nhctl.SuiteName(), cmd)
	if err != nil {
		return err
	}

	return profileGetWithJson(nhctl, "details", "helloWorld")
}

func ApplyCmForConfig(nhctl runner.Client, filePath *fp.FilePathEnhance) error {
	cmd := nhctl.GetNhctl().Command(
		context.TODO(), "apply", "bookinfo", filePath.Abs(),
	)

	return runner.Runner.RunSimple(
		nhctl.SuiteName(),
		cmd, false, func(s string) error {
			return nil
		},
	)
}

func ValidateImage(nhctl runner.Client, svcName string, svcType string, expectContain string) error {
	cmd := nhctl.GetNhctl().Command(
		context.TODO(), "profile", "get", "bookinfo", "-d", svcName, "-t", svcType,
		"--container", "xx",
		"--key", "image",
	)

	return runner.Runner.RunSimple(
		nhctl.SuiteName(),
		cmd, false, func(s string) error {
			if !strings.Contains(s, expectContain) {
				return errors.New(
					fmt.Sprintf(
						"profile not expected!! shold contain %s, now image %s", expectContain, s,
					),
				)
			}
			return nil
		},
	)
}

func ConfigReload(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(
		context.TODO(), "config",
		"reload", "bookinfo",
	)

	return runner.Runner.RunSimple(
		nhctl.SuiteName(),
		cmd, false, func(s string) error {
			return nil
		},
	)
}

func DeAssociate(nhctl runner.Client, svcName string, svcType string) error {
	cmd := nhctl.GetNhctl().Command(
		context.TODO(), "dev",
		"associate", "bookinfo",
		"-d", svcName, "-t", svcType, "--de-associate",
	)

	return runner.Runner.RunSimple(
		nhctl.SuiteName(),
		cmd, false, func(s string) error {
			return nil
		},
	)
}

func Associate(nhctl runner.Client, svcName string, svcType string, dir *fp.FilePathEnhance) error {
	cmd := nhctl.GetNhctl().Command(
		context.TODO(), "dev",
		"associate", "bookinfo",
		"-d", svcName, "-t", svcType, "--associate", dir.Abs(),
	)

	return runner.Runner.RunSimple(
		nhctl.SuiteName(),
		cmd, false, func(s string) error {
			return nil
		},
	)
}

func profileGetWithJson(nhctl runner.Client, container string, image string) error {
	tmp := &containerImage{}
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
	cmd := nhctl.GetNhctl().Command(
		ctx, "profile",
		"get", "bookinfo",
		"-d", "details", "-t", "deployment", "--container", container, "--key", "image",
	)
	stdout, stderr, err := runner.Runner.Run(nhctl.SuiteName(), cmd)
	if err != nil {
		return err
	}
	if stderr != "" {
		return errors.New(stderr)
	}
	if stdout == "" {
		return errors.New("profile get should be a json")
	}

	err = json.Unmarshal([]byte(stdout), tmp)
	if err != nil {
		return err
	}

	if image == "any" {
		if tmp.Image == "" {
			return errors.New("image of dev container config should not be nil")
		}
	} else {
		if tmp.Image != image {
			fmt.Printf("Image is %s\n", tmp.Image)
			return errors.New(fmt.Sprintf("image of dev container config should be %s", image))
		}
	}
	return nil
}
