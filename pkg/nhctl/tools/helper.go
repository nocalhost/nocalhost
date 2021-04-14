/*
Copyright 2020 The Nocalhost Authors.
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

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"math/rand"
	"nocalhost/pkg/nhctl/log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
)

var logger *zap.Logger

//check os
func CheckOS() string {
	sysType := runtime.GOOS
	switch sysType {
	case "linux":
	case "windows":
	case "darwin":
		break
	default:
		sysType = "unknown"
		break
	}
	return sysType
}

//check kubernetes version
func CheckK8s() (string, bool) {
	//osName := CheckOS()
	//if osName == "unknown" {
	//    panic("nonsupport os system. support linux, macos, windows.")
	//}
	////kubectl command
	//kubectl := osName+"/kubectl"
	//if osName == "windows" {
	//	kubectl = osName+"/kubectl.exe"
	//}

	kubectl := "kubectl"
	///home/coding-cli/
	//exec command
	command := exec.Command(kubectl, "version", "-o", "json")
	versionJson, err := command.CombinedOutput()
	if err != nil {
		fmt.Printf("cmds.Run() failed with %s\n", err)
	}
	var k8sVersion interface{}
	errVersion := json.Unmarshal(versionJson, &k8sVersion)
	if errVersion != nil {
		//logger.Error("Kubernetes api-server is not connected or Kubernetes is not installed", zap.Error(errVersion))
		fmt.Println("error: Kubernetes api-server is not connected or Kubernetes is not installed")
		return "", false
	}

	version := k8sVersion.(map[string]interface{})["serverVersion"].(map[string]interface{})["gitVersion"].(string)
	return version, true
}

//execute command
func ExecCommand(ctx context.Context, isDisplay bool, commandName string, params ...string) (string, error) {
	var errStdout, errStderr error
	var result []byte

	var cmd *exec.Cmd
	if ctx == nil {
		cmd = exec.Command(commandName, params...)
	} else {
		fmt.Println("command with ctx")
		cmd = exec.CommandContext(ctx, commandName, params...)
	}
	cmdStr := []string{commandName}
	cmdStr = append(cmdStr, params...)
	if isDisplay {
		log.Infof("cmd: %s", strings.Join(cmdStr, " "))
	}
	// log.Info(cmd.Args)
	stdoutIn, err := cmd.StdoutPipe()
	stderrIn, err2 := cmd.StderrPipe()
	if err != nil || err2 != nil {
		return "", errors.New("error: command failed to get stdout pipe")
	}
	err = cmd.Start()
	if err != nil {
		return "", errors.Wrap(err, "Failed to start cmd")
	}

	go func() {
		result, errStdout = copyAndCapture(os.Stdout, stdoutIn, isDisplay)
	}()

	go func() {
		_, errStderr = copyAndCapture(os.Stderr, stderrIn, isDisplay)
	}()

	err = cmd.Wait()
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	if errStderr != nil || errStdout != nil {
		log.Infof("%s %s", errStderr, errStdout)
		return "", errors.New("error occur when print")
	}

	return string(result), nil
}

func copyAndCapture(w io.Writer, r io.Reader, isDisplay bool) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			if isDisplay {
				w.Write(d)
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
	// never reached
	panic(true)
	return nil, nil
}

//file exist check
func CheckFile(file string) bool {
	_, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

//check cmds exist
func CheckCmdExists(cmd string) bool {
	path, err := exec.LookPath(cmd)
	if err != nil {
		logger.Error("check command is not exists ", zap.Error(err))
		return false
	} else {
		logger.Info("check command is exists ", zap.String("command", cmd), zap.String("path", path))
		return true
	}
}

// check kubectl and helm
func CheckThirdPartyCLI() (string, error) {
	kubectl := "kubectl"
	helm := "helm"
	if runtime.GOOS == "windows" {
		kubectl = "kubectl.exe"
		helm = "helm.exe"
	}
	_, err := exec.LookPath(kubectl)
	if err != nil {
		return "", err
	}
	_, err = exec.LookPath(helm)
	if err != nil {
		return "", err
	}
	return kubectl, nil
}

func GetNhctl() string {
	nhctl := "nhctl"
	if runtime.GOOS == "windows" {
		nhctl = "nhctl.exe"
	}
	return nhctl
}

func GenerateRangeNum(min, max int) int {
	rand.Seed(time.Now().Unix())
	randNum := rand.Intn(max-min) + min
	return randNum
}

func RemoveDuplicateElement(languages []string) []string {
	result := make([]string, 0, len(languages))
	temp := map[string]struct{}{}
	for _, item := range languages {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
