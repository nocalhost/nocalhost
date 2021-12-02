package testcase

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/test/runner"
	"nocalhost/test/util"
	"time"
)

// DeploymentReplaceAndDuplicate one replace one duplicate mode
func DeploymentReplaceAndDuplicate(cli runner.Client) {
	test(cli, "ratings", "deployment", profile.ReplaceDevMode)
}

// DeploymentDuplicateAndDuplicate two users into duplicate mode
func DeploymentDuplicateAndDuplicate(cli runner.Client) {
	test(cli, "ratings", "deployment", profile.DuplicateDevMode)
}

// StatefulsetReplaceAndDuplicate one replace one duplicate mode
func StatefulsetReplaceAndDuplicate(cli runner.Client) {
	test(cli, "web", "statefulset", profile.ReplaceDevMode)
}

// StatefulsetDuplicateAndDuplicate two users into duplicate mode
func StatefulsetDuplicateAndDuplicate(cli runner.Client) {
	test(cli, "web", "statefulset", profile.DuplicateDevMode)
}

func test(cli runner.Client, module string, moduleType string, devType profile.DevModeType) {
	port, _ := ports.GetAvailablePort()

	_ = PortForwardStartT(cli, module, moduleType, port)
	funcs := []func() error{
		//func() error { return PortForwardStartT(cli, module, moduleType, port) },
		func() error { return PortForwardCheck(port) },
		func() error { return StatusCheckPortForward(cli, module, moduleType, port) },
		func() error {
			if err := DevStartT(cli, module, moduleType, devType); err != nil {
				_ = DevEndT(cli, module, moduleType)
				return err
			}
			return nil
		},
		//func() error {
		//	util.Retry(fmt.Sprintf("[%s-%s-%s] PortForward", devType, module, moduleType), []func() error{
		//})
		//return nil
		//},
		func() error { return SyncCheckT(cli, module, moduleType) },
		func() error { return SyncStatusT(cli, module, moduleType) },
	}
	util.Retry(fmt.Sprintf("[%s-%s-%s] first user", devType, module, moduleType), funcs)
	util.Retry(
		fmt.Sprintf("[%s-%s-%s] do some magic operation", devType, module, moduleType), []func() error{
			func() error { return secretBackup(cli) },
			func() error { time.Sleep(time.Second * 5); return nil },
		},
	)

	//if devType.IsDuplicateDevMode() {
	//	secondPort, _ := ports.GetAvailablePort()
	//	_ = PortForwardStartT(cli, module, moduleType, secondPort)
	//	util.Retry(fmt.Sprintf("[%s-%s-%s] second user", devType, module, moduleType), []func() error{
	//		//func() error { return PortForwardStartT(cli, module, moduleType, secondPort) },
	//		func() error { return PortForwardCheck(secondPort) },
	//		func() error { return StatusCheckPortForward(cli, module, moduleType, secondPort) }},
	//	)
	//}
	util.Retry(
		fmt.Sprintf("[%s-%s-%s] second user", devType, module, moduleType), []func() error{
			func() error {
				if err := DevStartT(cli, module, moduleType, profile.DuplicateDevMode); err != nil {
					_ = DevEndT(cli, module, moduleType)
					return err
				}
				return nil
			},
			//func() error {
			//	util.Retry(fmt.Sprintf("[%s-%s-%s] PortForward again", devType, module, moduleType), []func() error{
			//	})
			//	return nil
			//},
			func() error { return SyncCheckT(cli, module, moduleType) },
			func() error { return SyncStatusT(cli, module, moduleType) },
		},
	)
	//util.Retry(fmt.Sprintf("[%s-%s-%s] second user check first user port-forward duplicate",
	//	devType, module, moduleType), []func() error{func() error { return PortForwardCheck(port) }},
	//)
	fmt.Printf("[%s-%s-%s] second user check exit duplicate", devType, module, moduleType)
	_ = DevEndT(cli, module, moduleType)
	// wait for 30 second for syncthing auto reconnect because syncthing will be killed if directory is the same
	time.Sleep(time.Second * 30)
	util.Retry(
		fmt.Sprintf("[%s-%s-%s] rollback secret", devType, module, moduleType),
		[]func() error{func() error { return secretRollback(cli) }},
	)
	//util.Retry(fmt.Sprintf("[%s-%s-%s] check first user's operation", devType, module, moduleType),
	//	[]func() error{
	//		func() error { return SyncCheckT(cli, module, moduleType) },
	//		func() error { return SyncStatusT(cli, module, moduleType) },
	//	},
	//)
	_ = DevEndT(cli, module, moduleType)
}

func secretBackup(cli runner.Client) error {
	secretName := appmeta.SecretNamePrefix + "bookinfo"

	secret, err := cli.GetClientset().CoreV1().Secrets(cli.GetNhctl().Namespace).
		Get(context.TODO(), secretName, v1.GetOptions{})
	if err != nil {
		return err
	}

	secret.Name = secretName + "backup"

	if _, err := cli.GetClientset().CoreV1().Secrets(cli.GetNhctl().Namespace).
		Create(context.TODO(), secret, v1.CreateOptions{}); err != nil {
		return err
	}

	return cli.GetClientset().CoreV1().Secrets(cli.GetNhctl().Namespace).
		Delete(context.TODO(), secretName, v1.DeleteOptions{})
}

func secretRollback(cli runner.Client) error {
	secretName := appmeta.SecretNamePrefix + "bookinfo"

	if err := cli.GetClientset().CoreV1().Secrets(cli.GetNhctl().Namespace).
		Delete(context.TODO(), secretName, v1.DeleteOptions{}); err != nil {
		return err
	}

	secret, err := cli.GetClientset().CoreV1().Secrets(cli.GetNhctl().Namespace).
		Get(context.TODO(), secretName+"backup", v1.GetOptions{})
	if err != nil {
		return err
	}

	secret.Name = appmeta.SecretNamePrefix

	if _, err := cli.GetClientset().CoreV1().Secrets(cli.GetNhctl().Namespace).
		Create(context.TODO(), secret, v1.CreateOptions{}); err != nil {
		return err
	}

	return err
}
