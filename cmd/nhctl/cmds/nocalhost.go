package cmds

import (
	"fmt"
	"io/ioutil"
	"os"
)

type NocalHost struct {
}

func (n *NocalHost) GetHomeDir() string {
	return fmt.Sprintf("%s%c%s", GetHomePath(), os.PathSeparator, DefaultNhctlHomeDirName)
}

func (n *NocalHost) GetApplicationDir() string {
	return fmt.Sprintf("%s%c%s", n.GetHomeDir(), os.PathSeparator, DefaultApplicationDirName)
}

func (n *NocalHost) GetApplications() ([]string, error) {
	appDir := n.GetApplicationDir()
	fs, err := ioutil.ReadDir(appDir)
	if err != nil {
		return nil, err
	}
	app := make([]string, 0)
	if fs == nil || len(fs) < 1 {
		return app, nil
	}
	for _, file := range fs {
		if file.IsDir() {
			app = append(app, file.Name())
		}
	}
	return app, err
}

func (n *NocalHost) CheckIfApplicationExist(appName string) bool {
	apps, err := n.GetApplications()
	if err != nil || apps == nil {
		return false
	}

	for _, app := range apps {
		if app == appName {
			return true
		}
	}

	return false
}

func (n *NocalHost) GetApplication(appName string) (*Application, error) {
	return NewApplication(appName)
}
