package daemon_handler

import (
	"context"
	"fmt"
	"io"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/dns"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/log"
	"os"
	"sync"
	"time"
)

// keep it in memory
var connected *pkg.ConnectOptions
var lock = &sync.Mutex{}

// HandleSudoVPNOperate sudo daemon, vpn executor
func HandleSudoVPNOperate(cmd *command.VPNOperateCommand, writer io.WriteCloser) error {
	preCheck(cmd)
	logCtx := util.GetContextWithLogger(writer)
	logger := util.GetLoggerFromContext(logCtx)
	connect := &pkg.ConnectOptions{
		Ctx:            logCtx,
		KubeconfigPath: cmd.KubeConfig,
		Namespace:      cmd.Namespace,
		Workloads:      []string{cmd.Resource},
	}
	if err := connect.InitClient(logCtx); err != nil {
		log.Error(util.EndSignFailed)
		writer.Close()
		return err
	}
	if err := connect.Prepare(logCtx); err != nil {
		log.Error(util.EndSignFailed)
		writer.Close()
		return err
	}
	switch cmd.Action {
	case command.Connect:
		lock.Lock()
		defer lock.Unlock()
		if connected != nil {
			logger.Errorf("already connected to namespace: %s\n", connected.Namespace)
			if connected.Namespace != cmd.Namespace {
				logger.Errorf("but want's to connect to: %s\n", cmd.Namespace)
			}
			logger.Infoln(util.EndSignFailed)
			writer.Close()
			return nil
		}
		connected = connect
		ctx, cancelFunc := context.WithCancel(context.TODO())
		remote.CancelFunctions = append(remote.CancelFunctions, cancelFunc)
		go func(namespace string, c *pkg.ConnectOptions, ctx context.Context) {
			// do until canceled
			for ctx.Err() == nil && c != nil {
				func() {
					defer func() {
						if err := recover(); err != nil {
							log.Error(err)
						}
					}()
					errChan, err := c.DoConnect(ctx)
					if err != nil {
						log.Warn(err)
						c.GetLogger().Infoln(util.EndSignFailed)
						time.Sleep(time.Second * 2)
						return
					}
					c.GetLogger().Infoln(util.EndSignOK)
					c.SetLogger(util.NewLogger(os.Stdout))
					// wait for exit
					if err = <-errChan; err != nil {
						fmt.Println(err)
						time.Sleep(time.Second * 2)
					}
				}()
			}
			// if exit
			//lock.Lock()
			//defer lock.Unlock()
			//logger.Info("prepare to exit, cleaning up")
			//dns.CancelDNS()
			//if c != nil {
			//	if err := c.ReleaseIP(); err != nil {
			//		logger.Errorf("failed to release ip to dhcp, err: %v", err)
			//	}
			//	remote.CleanUpTrafficManagerIfRefCountIsZero(c.GetClientSet(), namespace)
			//	logger.Info("clean up successful")
			//	connected = nil
			//}
			//remote.CancelFunctions = remote.CancelFunctions[:0]
			//return
		}(cmd.Namespace, connect, ctx)
	case command.DisConnect:
		// stop reverse resource
		// stop traffic manager
		lock.Lock()
		defer lock.Unlock()
		if connected == nil {
			logger.Infoln("already closed vpn")
			logger.Infoln(util.EndSignFailed)
			return nil
		}
		if !connected.IsSameKubeconfigAndNamespace(connect) {
			logger.Infoln("kubeconfig and namespace not match, can't disconnect vpn")
			logger.Infoln(util.EndSignFailed)
			return nil
		}

		for _, function := range remote.CancelFunctions {
			if function != nil {
				function()
			}
		}
		remote.CancelFunctions = remote.CancelFunctions[:0]
		//lock.Lock()
		//defer lock.Unlock()
		//for connected != nil {
		//	time.Sleep(time.Second * 2)
		//	logger.Info("wait for disconnect")
		//}
		logger.Info("prepare to exit, cleaning up")
		dns.CancelDNS()
		if err := connected.ReleaseIP(); err != nil {
			logger.Errorf("failed to release ip to dhcp, err: %v", err)
		}
		remote.CleanUpTrafficManagerIfRefCountIsZero(connected.GetClientSet(), connected.Namespace)
		logger.Info("clean up successful")
		connected = nil
		logger.Info(util.EndSignOK)
		return nil
	case command.Reconnect:
		logger.Info(util.EndSignOK)
		return nil
	}
	return nil
}

func init() {
	util.InitLogger(util.Debug)
	//go func() {
	//	if !util.IsAdmin() {
	//		return
	//	}
	//	for {
	//		var kubeConfigHost, namespace string
	//		if connected != nil {
	//			namespace = connected.Namespace
	//			kubeConfig, _ := clientcmd.RESTConfigFromKubeConfig(connected.KubeconfigBytes)
	//			if kubeConfig != nil {
	//				kubeConfigHost = kubeConfig.Host
	//			}
	//		}
	//		fmt.Printf("namespace: %s, kubeconfig: %s\n", namespace, kubeConfigHost)
	//		<-time.Tick(time.Second * 5)
	//	}
	//}()
}
