package daemon_handler

import (
	"context"
	"fmt"
	"io"
	"k8s.io/client-go/util/retry"
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

func HandleSudoVPNStatus() (interface{}, error) {
	return connected, nil
}

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
			if !connected.IsSameKubeconfigAndNamespace(connect) {
				logger.Errorf("already connected to namespace: %s, but want's to connect to namespace: %s\n",
					connected.Namespace, cmd.Namespace)
				logger.Infoln(util.EndSignFailed)
			} else {
				logger.Debugf("connected to spec cluster sucessufully")
				logger.Infoln(util.EndSignOK)
			}
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
						c.GetLogger().Errorln(err)
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
		}(cmd.Namespace, connect, ctx)
		return nil
	case command.DisConnect:
		// stop reverse resource
		// stop traffic manager
		lock.Lock()
		defer lock.Unlock()
		if connected == nil {
			logger.Infoln("already closed vpn")
			logger.Infoln(util.EndSignOK)
			return nil
		}
		// todo how to check it
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
		if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			return connected.ReleaseIP()
		}); err != nil {
			logger.Errorf("failed to release ip to dhcp, err: %v", err)
		}
		remote.CleanUpTrafficManagerIfRefCountIsZero(connected.GetClientSet(), connected.Namespace)
		logger.Info("clean up successful")
		connected = nil
		logger.Info(util.EndSignOK)
		return nil
	default:
		return nil
	}
}

func init() {
	util.InitLogger(util.Debug)
}
