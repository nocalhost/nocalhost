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
var connectNamespace string
var lock sync.Mutex

// HandleSudoVPNOperate sudo daemon, vpn executor
func HandleSudoVPNOperate(cmd *command.VPNOperateCommand, writer io.Writer) error {
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
		return err
	}
	if err := connect.Prepare(logCtx); err != nil {
		log.Error(util.EndSignFailed)
		return err
	}
	switch cmd.Action {
	case command.Connect:
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
					return
				default:
					errChan, err := connect.DoConnect(ctx)
					if err != nil {
						log.Warn(err)
						continue
					}
					c.Logger.Infoln(util.EndSignOK)
					// wait for exit
					<-errChan
				}
			}
		}(cmd.Namespace, connect)
	case command.DisConnect:
		//lock.Lock()
		//defer lock.Unlock()
		//if a.Load() == nil || !a.Load().(bool) {
		//	return nil
		//}
		// stop reverse resource
		// stop traffic manager
		for _, function := range remote.CancelFunctions {
			if function != nil {
				go function()
			}
		}
		logger.Info(util.EndSignOK)
	case command.Reconnect:

	case command.Reverse:
		//if a.Load() == nil {
		//	if _, err := connect.DoConnect(context.TODO()); err != nil {
		//		return err
		//	}
		//}
		if err := connect.DoReverse(context.TODO()); err != nil {
			logger.Errorln(err)
			logger.Info(util.EndSignFailed)
			return err
		}
		logger.Info(util.EndSignOK)
	case command.ReverseDisConnect:
		//if ok, err := IsBelongToMe(connect.GetClientSet().CoreV1().ConfigMaps(connect.Namespace), cmd.Resource); !ok || err != nil {
		//	return nil
		//}
		if err := connect.RemoveInboundPod(); err != nil {
			logger.Errorln(err)
			logger.Info(util.EndSignFailed)
			return err
		}
		logger.Info(util.EndSignOK)
	case command.Reset:

	}
	return nil
}

func init() {
	util.InitLogger(util.Debug)
}
