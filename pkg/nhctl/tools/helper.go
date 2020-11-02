package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var logger *zap.Logger

//init
//func init() {
//    logger = zaplog.InitLogger()
//}

//check os
func CheckOS() string{
	sysType := runtime.GOOS
	//logger.Info("OS name", zap.String("type", sysType))
	switch(sysType) {
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
func CheckK8s() (string,bool){
	osName := CheckOS()
    if osName == "unknown" {
        panic("nonsupport os system. support linux, macos, windows.")
    }
    //kubectl command
    kubectl := osName+"/kubectl"
    if osName == "windows" {
    	kubectl = osName+"/kubectl.exe"
    }

    ///home/coding-cli/
    //exec command
    command := exec.Command("utils/"+kubectl, "version", "-o", "json")
    versionJson, err := command.CombinedOutput()
    if err != nil {
        fmt.Printf("cmds.Run() failed with %s\n", err)
    }
    var k8sVersion interface{}
	errVersion := json.Unmarshal(versionJson, &k8sVersion)
	if errVersion != nil {
		//logger.Error("Kubernetes api-server is not connected or Kubernetes is not installed", zap.Error(errVersion))
		fmt.Println("error: Kubernetes api-server is not connected or Kubernetes is not installed")
		return "",false
	}

	//logger.Info("Kubernetes", zap.String("version", string(versionJson)))
	version := k8sVersion.(map[string]interface{})["serverVersion"].(map[string]interface{})["gitVersion"].(string)
    return version, true
}

func ExecKubeCtlCommand(ctx context.Context, kubeconfig string,  params ...string) error {
	var err error
	if kubeconfig != "" {
		params = append(params, "--kubeconfig")
		params = append(params, kubeconfig)
		_, err = ExecCommand(ctx, true,"kubectl", params...)
	} else {
		_, err = ExecCommand(ctx, true,"kubectl", params...)
	}
	return err
}

//execute command
func ExecCommand(ctx context.Context, isDisplay bool , commandName string,  params ...string) (string,error){
	//osName := CheckOS()
	//basePath := "utils/" + osName + "/"
	//execCommand := basePath + commandName
	execCommand := commandName

	// check command
	//if CheckCommand(commandName) {
	//	if !CheckFile(execCommand) {
	//		return "", errors.New("error: command not exists")
	//	}
	//}
	var cmd *exec.Cmd
	if ctx == nil {
		cmd = exec.Command(execCommand, params...)
	} else {
		fmt.Println("command with ctx")
		cmd = exec.CommandContext(ctx,execCommand, params...)
	}
	fmt.Println(cmd.Args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return "", errors.New("error: command failed to execute")
	}
	//start
	err = cmd.Start()
	if err != nil {
		fmt.Printf("failed to start cmds, err:%v\n", err)
		return "", err
	}
	reader := bufio.NewReader(stdout)
	//print output
	output := ""
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		output = output + line + "\n"
		if isDisplay {
			fmt.Print(line+"\n")
		}
	}
	err = cmd.Wait()
	if err != nil {
		return "", err
	}
	return output, nil
}


//Check Command
func CheckCommand(commandName string) bool{
	status := false
	toolsList := []string{
		"kubectl", 
		"kubectl.exe", 
		"mutagen", 
		"install-k8s-master", 
		"install-k8s-worker", 
		"install-k8s-create-kubeconfig", 
		"coding-deploy",
		"debug-service",
		"file-sync",
		"deploy-no-dependencies"}

	for _, v := range toolsList {
		if strings.Contains(commandName, v)	{
			status = true
			break
		}
	}
	return status
}


//file exist check
func CheckFile(file string) bool{
	_, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}


//check cmds exist
func CheckCmdExists(cmd string) bool{
    path, err := exec.LookPath(cmd)
    if err != nil {
        logger.Error("check command is not exists ", zap.Error(err))
        return false
    } else {
        logger.Info("check command is exists ", zap.String("command", cmd), zap.String("path", path))
        return true
    }
}


//check namespace
func CheckNS(namespace string) error{
	osName := CheckOS()
    if osName == "unknown" {
        panic("nonsupport os system. support linux, macos, windows.")
    }
    //kubectl command
    kubectl := osName+"/kubectl"
    if osName == "windows" {
    	kubectl = osName+"/kubectl.exe"
    }
    //exec command
	command := exec.Command("tools/"+kubectl, "get", "ns", "-o", "json")
    namespacesJson, err := command.CombinedOutput()
    if err != nil {
        fmt.Println("cmds.Run() failed with %s\n", err)
    }
    var namespaces interface{}
	errns := json.Unmarshal(namespacesJson, &namespaces)
	if errns != nil {
		logger.Error("get namespace failed", zap.Error(errns))
		fmt.Println("error: get namespace failed")
		return errors.New("get namespace failed")
	}

	logger.Info("Kubernetes", zap.String("namespaces", string(namespacesJson)))

	for _, v := range (namespaces.(map[string]interface{})["items"]).([]interface{}) {
		metadata := v.(map[string]interface{})["metadata"]
		ns := metadata.(map[string]interface{})["name"]
		if namespace == ns.(string) {
			logger.Error("namespace  exists ", zap.String("namespace", namespace))
			return errors.New("error: 此用户创建的namespace 已存在, 请更换用户名!")
		}
	}

    return nil
}











