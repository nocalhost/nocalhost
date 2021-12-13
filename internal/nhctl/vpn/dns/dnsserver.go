/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package dns

import (
	miekgdns "github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/cache"
	"strings"
	"time"
)

type server struct {
	// todo using cache to speed up dns resolve process
	dnsCache   *cache.LRUExpireCache
	forwardDNS *miekgdns.ClientConfig
	client     *miekgdns.Client
}

func NewDNSServer(network, address string, forwardDNS *miekgdns.ClientConfig) error {
	return miekgdns.ListenAndServe(address, network, &server{
		dnsCache:   cache.NewLRUExpireCache(1000),
		forwardDNS: forwardDNS,
		client: &miekgdns.Client{
			Net:          "udp",
			Timeout:      time.Second * 2,
			DialTimeout:  time.Second * 2,
			ReadTimeout:  time.Second * 2,
			WriteTimeout: time.Second * 2,
		},
	})
}

// ServeDNS consider using a cache
/*

nameserver 172.20.135.131
search nocalhost.svc.cluster.local svc.cluster.local cluster.local
options ndots:5

*/
func (s *server) ServeDNS(w miekgdns.ResponseWriter, r *miekgdns.Msg) {
	q := r.Question
	r.Question = make([]miekgdns.Question, 0, len(q))
	question := q[0]
	name := question.Name
	switch strings.Count(question.Name, ".") {
	case 1:
		question.Name = question.Name + s.forwardDNS.Search[0] + "."
	case 2:
		question.Name = question.Name + s.forwardDNS.Search[1] + "."
	case 3:
		question.Name = question.Name + s.forwardDNS.Search[2] + "."
	case 4:
		question.Name = question.Name + strings.Split(s.forwardDNS.Search[2], ".")[1] + "."
	case 5:
	default:
		w.Close()
		return
	}
	r.Question = []miekgdns.Question{question}
	answer, _, err := s.client.Exchange(r, s.forwardDNS.Servers[0]+":53")
	if err != nil {
		if !strings.Contains(err.Error(), "timeout") {
			log.Warnln(err)
		}
		if err = w.WriteMsg(r); err != nil {
			log.Warnln(err)
		}
	} else {
		if len(answer.Answer) != 0 {
			answer.Answer[0].Header().Name = name
		}
		if len(answer.Question) != 0 {
			answer.Question[0].Name = name
		}
		if err = w.WriteMsg(answer); err != nil {
			log.Warnln(err)
		}
	}
}
