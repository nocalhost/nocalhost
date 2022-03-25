package daemon_handler

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"k8s.io/client-go/util/retry"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/dns"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/log"
	"os"
	"runtime"
	"sync"
	"time"
)

// keep it in memory
var connected *pkg.ConnectOptions

//var done = make(chan struct{})
var lock = &sync.Mutex{}

func HandleSudoVPNStatus() (interface{}, error) {
	return connected, nil
}

// HandleSudoVPNOperate sudo daemon, vpn executor
func HandleSudoVPNOperate(cmd *command.VPNOperateCommand, writer io.WriteCloser) error {
	logCtx := util.GetContextWithLogger(writer)
	logger := util.GetLoggerFromContext(logCtx)
	connect := &pkg.ConnectOptions{
		Ctx:            logCtx,
		KubeconfigPath: cmd.KubeConfig,
		Namespace:      cmd.Namespace,
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
	once := &sync.Once{}
	switch cmd.Action {
	case command.Connect:
		lock.Lock()
		defer lock.Unlock()
		if !connected.IsEmpty() {
			if !connected.IsSameUid(connect) {
				logger.Errorf("connected to namespace: %s, but want's to connect to namespace: %s",
					connected.Namespace, cmd.Namespace)
				logger.Infoln(util.EndSignFailed)
			} else {
				//<-done
				if err := connected.WaitTrafficManagerToAssignAnIP(logger); err != nil {
					logger.Errorln(err)
					logger.Infoln(util.EndSignFailed)
				} else {
					logger.Debugf("connected to spec cluster sucessufully")
					logger.Infoln(util.EndSignOK)
				}
			}
			writer.Close()
			return nil
		}
		connected = connect
		ctx, cancelFunc := context.WithCancel(context.TODO())
		remote.CancelFunctions = append(remote.CancelFunctions, cancelFunc)
		go func(namespace string, options *pkg.ConnectOptions, ctx context.Context /*, c chan struct{}*/) {
			defer func() {
				if err := recover(); err != nil {
					disconnect(options.GetLogger())
					log.Error(err)
					runtime.Goexit()
				}
			}()
			// do until canceled
			for ctx.Err() == nil && options != nil {
				func() {
					errChan, err := options.DoConnect(ctx)
					if err != nil {
						options.GetLogger().Errorln(err)
						options.GetLogger().Infoln(util.EndSignFailed)
						disconnect(options.GetLogger())
						runtime.Goexit()
					}
					// judge if channel is already close
					//select {
					//case _, _ = <-c:
					//default:
					//	close(c)
					//}
					options.GetLogger().Infoln(util.EndSignOK)
					once.Do(func() { _ = writer.Close() })
					options.SetLogger(util.NewLogger(os.Stdout))
					// wait for exit
					if err = <-errChan; err != nil {
						fmt.Println(err)
						time.Sleep(time.Second * 2)
					}
					//c = make(chan struct{})
				}()
			}
		}(cmd.Namespace, connect, ctx /*, done*/)
		return nil
	case command.DisConnect:
		// stop reverse resource
		// stop traffic manager
		lock.Lock()
		defer lock.Unlock()
		defer writer.Close()
		if connected == nil {
			logger.Infoln("already closed vpn")
			logger.Infoln(util.EndSignOK)
			return nil
		}
		// todo how to check it
		if !connected.IsSameUid(connect) {
			logger.Infoln("kubeconfig and namespace not match, can't disconnect vpn")
			logger.Infoln(util.EndSignFailed)
			return nil
		}
		disconnect(logger)
		logger.Info(util.EndSignOK)
		return nil
	default:
		defer writer.Close()
		return fmt.Errorf("unsupported operation: %s", string(cmd.Action))
	}
}

func disconnect(logger *logrus.Logger) {
	for _, function := range remote.CancelFunctions {
		if function != nil {
			function()
		}
	}
	remote.CancelFunctions = remote.CancelFunctions[:0]
	logger.Info("prepare to exit, cleaning up")
	dns.CancelDNS()
	if connected != nil {
		if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			return connected.ReleaseIP()
		}); err != nil {
			logger.Errorf("failed to release ip to dhcp, err: %v", err)
		}
	}
	if connected != nil && connected.GetClientSet() != nil {
		remote.CleanUpTrafficManagerIfRefCountIsZero(connected.GetClientSet(), connected.Namespace)
	}
	logger.Info("clean up successful")
	connected = nil
	//done = make(chan struct{})
}

func init() {
	util.InitLogger(util.Debug)
}
