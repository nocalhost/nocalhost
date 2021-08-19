/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"context"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	"net"
	"os"
	"runtime"
	"time"
)

type esLog struct {
	Msg       string    `json:"msg"`
	PID       string    `json:"pid"`
	PPID      string    `json:"ppid"`
	Timestamp time.Time `json:"@timestamp"`
	Hostname  string    `json:"hostname"`
	Level     string    `json:"level"`
	Address   string    `json:"address"`
	Stack     string    `json:"stack,omitempty"`
	App       string    `json:"app,omitempty"`
	Arch      string    `json:"arch,omitempty"`
	Os        string    `json:"os,omitempty"`
	Line      string    `json:"line,omitempty"`
	Func      string    `json:"func,omitempty"`
	Version   string    `json:"version,omitempty"`
	Branch    string    `json:"branch,omitempty"`
	Commit    string    `json:"commit,omitempty"`
	Svc       string    `json:"svc,omitempty"`
}

var (
	esClient    *elastic.Client
	esProcessor *elastic.BulkProcessor
	esIndex     = "nocalhost"
	hostname    string
	address     string
)

func InitEs(host string) {

	var (
		err error
		ctx = context.TODO()
	)

	esClient, err = elastic.NewClient(elastic.SetSniff(false), elastic.SetURL(host))
	if err != nil {
		esClient = nil
		return
	}

	// Create Index if not exist
	nhIndexMapping := `
{
  "mappings": {
    "properties": {
	"@timestamp": {
        "type": "date"
      },
      "msg": {
        "type": "text"
      },
      "address": {
        "type": "text"
      },
      "pid": {
        "type": "text"
      },
      "ppid": {
        "type": "text"
      },
	  "hostname": {
        "type": "text"
      },
	  "app": {
        "type": "text"
      },
	  "svc": {
        "type": "text"
      },
	  "commit": {
        "type": "text"
      },
	  "branch": {
        "type": "text"
      },
	  "version": {
        "type": "text"
      },
	  "line": {
        "type": "text"
      },
	  "func": {
        "type": "text"
      },
	  "os": {
        "type": "text"
      },
	  "arch": {
        "type": "text"
      },
	  "stack": {
        "type": "text"
      },
      "level": {
        "type": "text"
      }
    }
  }
}`

	exists, err := esClient.IndexExists(esIndex).Do(ctx)
	if err != nil {
		return
	}

	if !exists {
		createIndex, err := esClient.CreateIndex(esIndex).BodyJson(nhIndexMapping).Do(ctx)
		if err != nil {
			esClient = nil
			return
		}
		if !createIndex.Acknowledged {
			esClient = nil
			return
		}
	}
	hostname, _ = os.Hostname()

	ip, err := externalIP()
	if err == nil && ip != nil {
		address = ip.String()
	}

	esProcessor, err = esClient.BulkProcessor().
		Name(hostname).
		BulkActions(-1).
		BulkSize(-1).
		FlushInterval(100 * time.Millisecond).
		Do(ctx)
	if err != nil {
		fmt.Println("proccessor created failed", err.Error())
	}
}

func writeStackToEs(level string, msg string, stack string) {
	if esClient == nil {
		return
	}

	var (
		lineNum string
		funName string
	)
	funcName, file, line, ok := runtime.Caller(2)
	if ok {
		funName = runtime.FuncForPC(funcName).Name()
		lineNum = fmt.Sprintf("%s:%d", file, line)
	}

	write := func() {
		data := esLog{
			Msg:       msg,
			PID:       fields["PID"],
			PPID:      fields["PPID"],
			App:       fields["APP"],
			Svc:       fields["SVC"],
			Version:   fields["VERSION"],
			Commit:    fields["COMMIT"],
			Branch:    fields["BRANCH"],
			Timestamp: time.Now(),
			Hostname:  hostname,
			Level:     level,
			Address:   address,
			Stack:     stack,
			Os:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			Line:      lineNum,
			Func:      funName,
		}
		esClient.Index().Index(esIndex).BodyJson(&data).Refresh("true").Do(context.Background())
	}

	if os.Getenv("NOCALHOST_TRACE") != "" {
		write()
	} else {
		go func() {
			write()
		}()
	}
}

func externalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip, nil
		}
	}
	return nil, errors.New("connected to the network?")
}

func getIpFromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil || ip.IsLoopback() {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil // not an ipv4 address
	}

	return ip
}
