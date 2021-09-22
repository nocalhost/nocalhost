/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package log

import (
	"context"
	"encoding/json"
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
	Bulk      bool      `json:"bulk"`
	Args      string    `json:"args,omitempty"`
}

var (
	esClient    *elastic.Client
	esProcessor *elastic.BulkProcessor
	esIndex     = "nocalhost"
	hostname    string
	address     string
	useBulk     bool
)

// UseBulk Used by daemon
func UseBulk(enable bool) {
	useBulk = enable
}

func InitEs(host string) {

	var (
		err error
		ctx = context.TODO()
	)

	esClient, err = elastic.NewClient(elastic.SetSniff(false), elastic.SetURL(host),
		elastic.SetHealthcheckTimeoutStartup(1*time.Second))
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
      "args": {
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
		//BulkActions(-1).
		//BulkSize(-1).
		FlushInterval(10 * time.Second).
		Do(ctx)
	if err != nil {
		fmt.Println("proccessor created failed", err.Error())
	}
}

func writeStackToEs(level string, msg string, stack string) {
	writeStackToEsWithField(level, msg, stack, nil)
}

func writeStackToEsWithField(level string, msg string, stack string, field map[string]interface{}) {
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
			Args:      fields["ARGS"],
			Timestamp: time.Now(),
			Hostname:  hostname,
			Level:     level,
			Address:   address,
			Stack:     stack,
			Os:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			Line:      lineNum,
			Func:      funName,
			Bulk:      useBulk,
		}

		mapping := make(map[string]interface{}, 0)
		if mas, err := json.Marshal(data); err == nil {
			if json.Unmarshal(mas, &mapping) != nil {
				return
			}
		}

		for k, v := range field {
			mapping[k] = v
		}

		if useBulk && esProcessor != nil {
			esProcessor.Add(elastic.NewBulkIndexRequest().Index(esIndex).Doc(&mapping))
		} else {
			esClient.Index().Index(esIndex).BodyJson(&mapping).Refresh("true").Do(context.Background())
		}
	}

	if os.Getenv("NOCALHOST_TRACE") != "" && !useBulk {
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
