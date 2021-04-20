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

package nhctlcli

import (
	"context"
	"os/exec"
)

type CLI struct {
	KubeConfig string
	Namespace  string
}

type Config interface {
	GetKubeConfig() string
	GetNamespace() string
}

func NewCLI(cfg Config, defaultNamespace string) *CLI {
	namespace := defaultNamespace
	if kubeNamespace := cfg.GetNamespace(); kubeNamespace != "" {
		namespace = kubeNamespace
	}
	return &CLI{
		KubeConfig: cfg.GetKubeConfig(),
		Namespace:  namespace,
	}
}

func (c *CLI) Command(ctx context.Context, command string, arg ...string) *exec.Cmd {
	args := c.argsAppendNamespaceAndKubeconfig(command, "", arg...)
	return exec.CommandContext(ctx, "nhctl", args...)
}

func (c *CLI) CommandWithNamespace(ctx context.Context, command string, namespace string, arg ...string) *exec.Cmd {
	args := c.argsAppendNamespaceAndKubeconfig(command, namespace, arg...)
	return exec.CommandContext(ctx, "nhctl", args...)
}

func (c CLI) Run(ctx context.Context, command string, arg ...string) (string, string, error) {
	cmd := c.Command(ctx, command, arg...)
	return Runner.Run(cmd)
}

func (c CLI) RunWithRollingOut(ctx context.Context, command string, arg ...string) (string, string, error) {
	cmd := c.Command(ctx, command, arg...)
	return Runner.RunWithRollingOut(cmd)
}

func (c *CLI) argsAppendNamespaceAndKubeconfig(command string, namespace string, arg ...string) []string {
	var args []string
	namespace = c.getNamespace(namespace)
	if namespace != "" {
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
