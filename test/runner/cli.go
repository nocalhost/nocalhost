/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package runner

import (
	"context"
	"os/exec"
)

type CLI struct {
	KubeConfig string
	Namespace  string
	Cmd        string
	cfg        Config
}

type Config interface {
	GetKubeConfig() string
	GetNamespace() string
	GetCmd() string
	SuitName() string
}

func NewCLI(cfg Config, defaultNamespace string) *CLI {
	namespace := defaultNamespace
	if kubeNamespace := cfg.GetNamespace(); kubeNamespace != "" {
		namespace = kubeNamespace
	}
	return &CLI{
		KubeConfig: cfg.GetKubeConfig(),
		Namespace:  namespace,
		Cmd:        cfg.GetCmd(),
		cfg:        cfg,
	}
}

func (c *CLI) Command(ctx context.Context, command string, arg ...string) *exec.Cmd {
	args := c.argsAppendNamespaceAndKubeconfig(false, command, "", arg...)
	return exec.CommandContext(ctx, c.Cmd, args...)
}

func (c *CLI) CommandWithoutNs(ctx context.Context, command string, arg ...string) *exec.Cmd {
	args := c.argsAppendNamespaceAndKubeconfig(true, command, "", arg...)
	return exec.CommandContext(ctx, c.Cmd, args...)
}

func (c *CLI) CommandWithNamespace(ctx context.Context, command string, namespace string, arg ...string) *exec.Cmd {
	args := c.argsAppendNamespaceAndKubeconfig(false, command, namespace, arg...)
	return exec.CommandContext(ctx, c.Cmd, args...)
}

func (c CLI) SuitName() string {
	return c.cfg.SuitName()
}

func (c CLI) Run(ctx context.Context, command string, arg ...string) (string, string, error) {
	cmd := c.Command(ctx, command, arg...)
	return Runner.Run(c.SuitName(), cmd)
}

func (c CLI) RunClusterScope(ctx context.Context, command string, arg ...string) (string, string, error) {
	cmd := c.CommandWithoutNs(ctx, command, arg...)
	return Runner.Run(c.SuitName(), cmd)
}

func (c CLI) RunWithRollingOut(ctx context.Context, command string, arg ...string) (string, string, error) {
	cmd := c.Command(ctx, command, arg...)
	return Runner.RunWithRollingOutWithChecker(c.SuitName(), cmd, nil)
}

func (c *CLI) argsAppendNamespaceAndKubeconfig(clusterScope bool, command string, namespace string, arg ...string) []string {
	var args []string
	namespace = c.getNamespace(namespace)
	if namespace != "" && !clusterScope {
		args = append(args, "--namespace", namespace)
	}
	if c.KubeConfig != "" {
		args = append(args, "--kubeconfig", c.KubeConfig)
	}
	args = append(args, command)
	args = append(args, arg...)
	return args
}

func (c *CLI) getNamespace(defaultValue string) string {
	if defaultValue != "" {
		return defaultValue
	}
	return c.Namespace
}
