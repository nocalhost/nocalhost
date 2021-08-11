/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package util

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"time"
)

func TimeoutFunc(d time.Duration, do, compensating func() error) error {
	ctx, _ := context.WithTimeout(context.Background(), d)

	errorChan := make(chan error, 1)
	go func() {
		errorChan <- do()
	}()

	select {
	case <-ctx.Done():

		if compensating != nil {
			if err := compensating(); err != nil {
				return errors.Wrap(err, "Exec Timeout! And compensating Error! ")
			}
		}

		return errors.New("Exec Timeout!")
	case err := <-errorChan:

		if compensating != nil && err != nil {
			if e := compensating(); e != nil {
				return errors.Wrap(e, fmt.Sprintf("Exec Fail, Error %s! And compensating Error! ", err))
			}
		}

		return err
	}
}

func TimeoutChecker(d time.Duration, cancanFunc func()) {
	tick := time.Tick(d)
	for {
		select {
		case <-tick:
			if cancanFunc != nil {
				cancanFunc()
			}
			panic(fmt.Sprintf("test case failed, timeout: %v", d))
		}
	}
}

func NeedsToInitK8sOnTke() bool {
	debug := os.Getenv(Local)
	if debug != "" {
		return false
	}
	return true
	//if strings.Contains(runtime.GOOS, "darwin") {
	//	return true
	//} else if strings.Contains(runtime.GOOS, "windows") {
	//	return true
	//} else {
	//	return false
	//}
}

func GetKubeconfig() string {
	kubeconfig := os.Getenv(KubeconfigPath)
	if kubeconfig == "" {
		dir, _ := os.UserHomeDir()
		kubeconfig = filepath.Join(dir, ".kube", "config")
	}
	return kubeconfig
}
