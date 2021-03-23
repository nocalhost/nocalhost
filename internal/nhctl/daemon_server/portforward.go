/*
Copyright 2021 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package daemon_server

//var (
//	dbPortForwardLocker sync.Mutex
//)

//func checkLocalPortStatus(ctx context.Context, svc *model.NocalHostResource, sLocalPort, sRemotePort int) {
//	for {
//		select {
//		case <-ctx.Done():
//			log.Logf("Stop Checking port %d:%d's status", sLocalPort, sRemotePort)
//			//_ = a.UpdatePortForwardStatus(deployment, sLocalPort, sRemotePort, portStatus, "Stopping")
//			return
//		default:
//			var portStatus string
//			available := ports.IsTCP4PortAvailable("127.0.0.1", sLocalPort)
//			if available {
//				portStatus = "CLOSED"
//			} else {
//				portStatus = "LISTEN"
//			}
//			log.Infof("Checking Port %d:%d's status: %s", sLocalPort, sRemotePort, portStatus)
//
//			err := updatePortForwardStatus(svc, sLocalPort, sRemotePort, portStatus, "Check local port status")
//			if err != nil {
//				log.LogE(err)
//			} else {
//				log.Logf("Port-forward %d:%d's status updated", sLocalPort, sRemotePort)
//			}
//			<-time.After(2 * time.Minute)
//		}
//	}
//}
