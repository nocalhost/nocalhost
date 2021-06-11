package base

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/pkg/nhctl/log"
	"strings"
)

type SvcType string

const (
	Deployment  SvcType = "deployment"
	StatefulSet SvcType = "statefulset"
	DaemonSet   SvcType = "daemonset"
	Job         SvcType = "job"
	CronJob     SvcType = "cronjob"

	DEPLOYMENT SvcType = "D"
)

func SvcTypeOf(svcType string) SvcType {
	serviceType := Deployment
	if svcType != "" {
		svcTypeLower := strings.ToLower(svcType)
		switch svcTypeLower {
		case strings.ToLower(string(Deployment)):
			serviceType = Deployment
		case strings.ToLower(string(StatefulSet)):
			serviceType = StatefulSet
		case strings.ToLower(string(DaemonSet)):
			serviceType = DaemonSet
		case strings.ToLower(string(Job)):
			serviceType = Job
		case strings.ToLower(string(CronJob)):
			serviceType = CronJob
		default:
			log.FatalE(errors.New(fmt.Sprintf("Unsupported SvcType %s", svcType)), "")
		}
	}
	return serviceType
}

// Alias For compatibility with meta
func (s SvcType) Alias() SvcType {
	if s == Deployment {
		return DEPLOYMENT
	}
	return s
}

func (s SvcType) Origin() SvcType {
	if s == DEPLOYMENT {
		return Deployment
	}
	return s
}

func (s SvcType) String() string {
	return string(s)
}
