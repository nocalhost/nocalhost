package daemon_handler

import (
	"context"
	"io"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/dns"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/log"
	"sync"
)

// keep it in memory
var connected *pkg.ConnectOptions
var lock sync.Mutex

// HandleSudoVPNOperate sudo daemon, vpn executor
func HandleSudoVPNOperate(cmd *command.VPNOperateCommand, writer io.WriteCloser) error {
	logCtx := util.GetContextWithLogger(writer)
	logger := util.GetLoggerFromContext(logCtx)
	connect := &pkg.ConnectOptions{
		Logger:         logger,
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
			logger.Errorln("already connected")
			logger.Infoln(util.EndSignFailed)
			writer.Close()
			return nil
		}
		connected = connect
		ctx, cancelFunc := context.WithCancel(context.TODO())
		remote.CancelFunctions = append(remote.CancelFunctions, cancelFunc)
		go func(namespace string, c *pkg.ConnectOptions) {
			for {
				select {
				case <-ctx.Done():
					logger.Info("prepare to exit, cleaning up")
					dns.CancelDNS()
					if err := c.ReleaseIP(); err != nil {
						logger.Errorf("failed to release ip to dhcp, err: %v", err)
					}
					remote.CleanUpTrafficManagerIfRefCountIsZero(c.GetClientSet(), namespace)
					logger.Info("clean up successful")
					connected = nil
					return
				default:
					errChan, err := connect.DoConnect(ctx)
					if err != nil {
						log.Warn(err)
						c.Logger.Infoln(util.EndSignFailed)
						continue
					}
					c.Logger.Infoln(util.EndSignOK)
					// wait for exit
					<-errChan
				}
			}
		}(cmd.Namespace, connect)
	case command.DisConnect:
		// stop reverse resource
		// stop traffic manager
		for _, function := range remote.CancelFunctions {
			if function != nil {
				go function()
			}
		}
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
	case command.Reconnect:
	}
	return nil
}

func init() {
	util.InitLogger(util.Debug)
}
