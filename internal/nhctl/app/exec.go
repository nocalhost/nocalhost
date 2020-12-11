package app

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"nocalhost/pkg/nhctl/log"
)

func (a *Application) EnterPodTerminal(svcName string) error {
	podList, err := a.client.GetPodsFromDeployment(context.TODO(), a.GetNamespace(), svcName)
	if err != nil {
		return err
	}
	if len(podList.Items) != 1 {
		log.Warnf("the number of pods of %s is not 1 ???", svcName)
		return errors.New(fmt.Sprintf("the number of pods of %s is not 1 ???", svcName))
	}
	pod := podList.Items[0].Name
	return a.client.ExecBash(a.GetNamespace(), pod, "")
}

func (a *Application) Exec(svcName string, commands []string) error {
	podList, err := a.client.GetPodsFromDeployment(context.TODO(), a.GetNamespace(), svcName)
	if err != nil {
		return err
	}
	if len(podList.Items) != 1 {
		log.Warnf("the number of pods of %s is not 1 ???", svcName)
		return errors.New(fmt.Sprintf("the number of pods of %s is not 1 ???", svcName))
	}
	pod := podList.Items[0].Name
	return a.client.Exec(a.GetNamespace(), pod, "", commands)
}
