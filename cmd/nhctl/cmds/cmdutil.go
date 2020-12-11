package cmds

import (
	"github.com/sirupsen/logrus"

	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"
)

func InitApp(appName string) {
	var err error
	if settings.Debug {
		log.SetLevel(logrus.DebugLevel)
	}

	if !nh.CheckIfApplicationExist(appName) {
		log.Fatalf("application \"%s\" not found", appName)
	}
	nocalhostApp, err = app.NewApplication(appName)
	if err != nil {
		log.Fatal("failed to get application info")
	}
}

func CheckIfSvcExist(svcName string) {
	if svcName == "" {
		log.Fatal("please use -d to specify a k8s workload")
	}

	exist, err := nocalhostApp.CheckIfSvcExist(svcName, app.Deployment)
	if err != nil {
		log.Fatalf("failed to check if svc exists : %v", err)
	} else if !exist {
		log.Fatalf("\"%s\" not found", svcName)
	}

}

func InitAppAndCheckIfSvcExist(appName string, svcName string) {
	InitApp(appName)
	CheckIfSvcExist(svcName)
}
