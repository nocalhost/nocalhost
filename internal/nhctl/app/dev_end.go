package app

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/pkg/nhctl/log"
	"runtime"
)

func (a *Application) EndDevelopMode(svcName string, fileSyncOps *FileSyncOptions) error {
	var err error
	if !a.CheckIfSvcIsDeveloping(svcName) {
		errors.New(fmt.Sprintf("\"%s\" is not developing status", svcName))
	}

	fmt.Println("ending DevMode...")
	// end file sync
	fmt.Println("terminating file sync process...")
	// get free port recorded in dev-start stage so we don't need to get free port again
	var devStartOptions = &DevStartOptions{}
	fileSyncOps, err = a.GetSyncthingPort(svcName, fileSyncOps)
	if err != nil {
		log.Warnf("failed to get syncthing port, you can start sync command first. error message: %s \n", err.Error())
		return err
	}

	newSyncthing, err := a.NewSyncthing(svcName, devStartOptions, fileSyncOps)
	if err != nil {
		log.Warnf("failed to start syncthing process: %s", err.Error())
		return err
	}
	// read and empty pid file
	portForwardPid, portForwardFilePath, err := a.GetBackgroundSyncPortForwardPid(svcName, false)
	if err != nil {
		log.Warnf("failed to get background port-forward pid file, ignored")
	}
	if portForwardPid != 0 {
		err = newSyncthing.Stop(portForwardPid, portForwardFilePath, "port-forward", true)
		if err != nil {
			log.Warnf("fail stop port-forward progress pid %d, please run `kill -9 %d` by manual, err: %s\n", portForwardPid, portForwardPid, err)
		}
	}

	// read and clean pid file
	syngthingPid, syncThingPath, err := a.GetBackgroundSyncThingPid(svcName, false)
	if err != nil {
		log.Warn("failed to get background port-forward pid file, ignored")
	}
	if syngthingPid != 0 {
		err = newSyncthing.Stop(syngthingPid, syncThingPath, "syncthing", true)
		if err != nil {
			if runtime.GOOS == "windows" {
				// in windows, it will raise a "Access is denied" err when killing progress, so we cab ignore this err
				fmt.Printf("attempt to terminate syncthing process(pid: %d), you can run `tasklist | findstr %d` to make sure process was exited\n", portForwardPid, portForwardPid)
			} else {
				log.Warnf("failed to terminate syncthing process(pid: %d), please run `kill -9 %d` manually, err: %s\n", portForwardPid, portForwardPid, err)
			}
		}
	}

	if err == nil { // none of them has error
		fmt.Printf("background port-forward process: %d and  syncthing process: %d terminated.\n", portForwardPid, syngthingPid)
	}

	// end dev port background port forward
	// read and empty pid file
	onlyPortForwardPid, onlyPortForwardFilePath, err := a.GetBackgroundOnlyPortForwardPid(svcName, false)
	if err != nil {
		fmt.Println("no dev port-forward pid file found, ignored.")
	}
	if onlyPortForwardPid != 0 {
		err = newSyncthing.Stop(onlyPortForwardPid, onlyPortForwardFilePath, "port-forward", true)
		if err != nil {
			fmt.Printf("[info] failed to terminate dev port-forward process(pid %d), please run `kill -9 %d` manually\n", onlyPortForwardPid, onlyPortForwardPid)
		}
	}

	if err == nil {
		fmt.Printf("dev port-forward: %d has been ended\n", onlyPortForwardPid)
	}

	// set profile status
	// set port-forward port and ignore result
	_ = a.SetSyncthingPort(svcName, 0, 0, 0, 0)
	// roll back workload
	log.Debug("rolling back workload...")
	err = a.RollBack(context.TODO(), svcName)
	if err != nil {
		fmt.Printf("failed to rollback.\n")
		return err
	}
	err = a.SetDevEndProfileStatus(svcName)
	if err != nil {
		log.Warn("failed to update \"developing\" status")
		return err
	}
	return nil
}
