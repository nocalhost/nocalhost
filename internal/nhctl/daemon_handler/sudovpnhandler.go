package daemon_handler

import (
	"context"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/dns"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/pkg/nhctl/log"
	"sync"
	"sync/atomic"
)

var a atomic.Value
var lock sync.Mutex

// HandleSudoVPNOperate sudo daemon, vpn executor
func HandleSudoVPNOperate(cmd *command.VPNOperateCommand) error {
	switch cmd.Action {
	case command.Connect:
		lock.Lock()
		defer lock.Unlock()
		if a.Load() != nil && a.Load().(bool) {
			return nil
		}
		a.Store(true)
		connect := &pkg.ConnectOptions{
			KubeconfigPath: cmd.KubeConfig,
			Namespace:      cmd.Namespace,
			Workloads:      []string{cmd.Resource},
		}
		ctx, cancelFunc := context.WithCancel(context.TODO())
		remote.CancelFunctions = append(remote.CancelFunctions, cancelFunc)
		go func(namespace string, c *pkg.ConnectOptions, a *atomic.Value) {
			for {
				select {
				case <-ctx.Done():
					log.Info("prepare to exit, cleaning up")
					dns.CancelDNS()
					if err := c.ReleaseIP(); err != nil {
						log.Errorf("failed to release ip to dhcp, err: %v", err)
					}
					remote.CleanUpTrafficManagerIfRefCountIsZero(c.GetClientSet(), namespace)
					log.Info("clean up successful")
					a.Store(false)
					return
				default:
					if err := connect.InitClient(); err != nil {
						return
					}
					errChan, err := connect.DoConnect(ctx)
					if err != nil {
						log.Warn(err)
						continue
					}
					// wait for exit
					<-errChan
				}
			}
		}(cmd.Namespace, connect, &a)
	case command.DisConnect:
		lock.Lock()
		defer lock.Unlock()
		if a.Load() == nil || !a.Load().(bool) {
			return nil
		}
		// stop reverse resource
		if len(cmd.Resource) != 0 {

		} else // stop traffic manager
		{
			for _, function := range remote.CancelFunctions {
				if function != nil {
					go function()
				}
			}
		}
	case command.Reconnect:
	case command.Reset:

	}
	return nil
}
