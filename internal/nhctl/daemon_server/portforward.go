/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package daemon_server

//var (
//	dbPortForwardLocker sync.Mutex
//)

//func checkLocalPortStatus(ctx context.Context, controller *model.NocalHostResource, sLocalPort, sRemotePort int) {
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
//			err := updatePortForwardStatus(controller, sLocalPort, sRemotePort, portStatus, "Check local port status")
//			if err != nil {
//				log.LogE(err)
//			} else {
//				log.Logf("Port-forward %d:%d's status updated", sLocalPort, sRemotePort)
//			}
//			<-time.After(2 * time.Minute)
//		}
//	}
//}
