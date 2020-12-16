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

package daemon

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const MARK_ENV_NAME = "NH_SYNC_DAEMON_IDX"

// run background times
var runIdx = 0

type Daemon struct {
	LogFile     string
	PidFile     string
	MaxCount    int
	MaxError    int
	MinExitTime int64
}

// start a child process
// if isExit is true, exit parent process itself
func Background(logFile, pidFile string, isExit bool) (*exec.Cmd, error) {
	runIdx++
	// check if this process is a child process
	envIdx, err := strconv.Atoi(os.Getenv(MARK_ENV_NAME))
	if err != nil {
		envIdx = 0
	}
	if runIdx <= envIdx { // this is already a child process, exit
		return nil, nil
	}

	// set env for child process
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=%d", MARK_ENV_NAME, runIdx))

	// start child process
	cmd, err := startProc(os.Args, env, logFile)
	if err != nil {
		log.Println(os.Getpid(), "run background fail:", err)
		return nil, err
	} else {
		// write pid file to application dir
		log.Println(os.Getpid(), ":", "run background success, pid: ", "->", cmd.Process.Pid, "\n ")
		file, err := os.OpenFile(pidFile, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		sPid := strconv.Itoa(cmd.Process.Pid)
		_, err = file.Write([]byte(sPid))
		if err != nil {
			return nil, err
		}
	}

	if isExit {
		os.Exit(0)
	}

	return cmd, nil
}

func NewDaemon(logFile string) *Daemon {
	return &Daemon{
		LogFile:     logFile,
		MaxCount:    0,
		MaxError:    3,
		MinExitTime: 10,
	}
}

// start daemon
func (d *Daemon) Run() {
	Background(d.LogFile, d.PidFile, true)
	var t int64
	count := 1
	errNum := 0
	for {
		dInfo := fmt.Sprintf("daemon(pid:%d; count:%d/%d; errNum:%d/%d):",
			os.Getpid(), count, d.MaxCount, errNum, d.MaxError)
		if errNum > d.MaxError {
			log.Println(dInfo, "daemon fail too much, exit")
			os.Exit(1)
		}
		if d.MaxCount > 0 && count > d.MaxCount {
			log.Println(dInfo, "daemon restart too much, exit")
			os.Exit(0)
		}
		count++

		t = time.Now().Unix()
		cmd, err := Background(d.LogFile, d.PidFile, false)
		if err != nil { // start fail
			log.Println(dInfo, "child progress run fail;", "err:", err)
			errNum++
			continue
		}

		// child progress,
		if cmd == nil {
			log.Printf("child pid=%d: started", os.Getpid())
			break
		}

		//father wait
		err = cmd.Wait()
		dat := time.Now().Unix() - t
		if dat < d.MinExitTime { // fail and exit
			errNum++
		} else { // normal exit
			errNum = 0
		}
		log.Printf("%s child (%d) progress exit, tootal run %d second: %v\n", dInfo, cmd.ProcessState.Pid(), dat, err)
	}
}

func startProc(args, env []string, logFile string) (*exec.Cmd, error) {
	cmd := &exec.Cmd{
		Path:        args[0],
		Args:        args,
		Env:         env,
		SysProcAttr: NewSysProcAttr(),
	}

	if logFile != "" { // child progress might not have permission
		stdout, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Println(os.Getpid(), ": open log file err :", err)
			return nil, err
		}
		cmd.Stderr = stdout
		cmd.Stdout = stdout
	}

	// TODO if will success in windows?
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}
