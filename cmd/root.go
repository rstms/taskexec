/*
Copyright © 2025 Matt Krueger <mkrueger@rstms.net>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

 1. Redistributions of source code must retain the above copyright notice,
    this list of conditions and the following disclaimer.

 2. Redistributions in binary form must reproduce the above copyright notice,
    this list of conditions and the following disclaimer in the documentation
    and/or other materials provided with the distribution.

 3. Neither the name of the copyright holder nor the names of its contributors
    may be used to endorse or promote products derived from this software
    without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var exitCode int

const DEFAULT_TASK_NAME = "b62a95c5-3b0e-4c3d-aceb-fdf20308e3c3"

var rootCmd = &cobra.Command{
	Version: "0.0.4",
	Use:     "taskexec COMMAND [ARG]...",
	Short:   "execute command using windows task scheduler",
	Long: `
If running on windows, invoke schtasks.exe to /CREATE a task for the command,
/RUN the task, then /DELETE the task.
This mechanism should allow GUI programs to be started from a client session
on the windows OpenSSH daemon.
On non-windows systems, the arguments are executed with the shell defined
in the SHELL environment variable, defaulting to /bin/sh if SHELL is not set
`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := TaskExec(strings.Join(args, " "))
		cobra.CheckErr(err)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
	os.Exit(exitCode)
}
func init() {
	cobra.OnInitialize(InitConfig)
	OptionString("logfile", "l", "", "log filename")
	OptionString("config", "c", "", "config file")
	OptionSwitch("debug", "", "produce debug output")
	OptionSwitch("verbose", "v", "increase verbosity")
	OptionString("taskname", "t", "", "task name")
}

func TaskExec(command string) error {
	if runtime.GOOS == "windows" {
		return WinTaskExec(command)
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	fmt.Printf("command: %s\n", command)
	cmd := exec.Command(shell, "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	switch err.(type) {
	case *exec.ExitError:
		exitCode = cmd.ProcessState.ExitCode()
		err = nil
	}
	return err
}

func DeleteTask(name string) error {
	return exec.Command("schtasks.exe", "/delete", "/tn", name, "/f").Run()
}

func WinTaskExec(command string) error {
	name := viper.GetString("taskname")
	if name == "" {
		name = DEFAULT_TASK_NAME
	}

	DeleteTask(name)

	cmd := exec.Command("schtasks.exe", "/create", "/tn", name, "/sc", "onstart", "/it", "/tr", command)
	err := cmd.Run()
	cobra.CheckErr(err)

	defer DeleteTask(name)

	cmd = exec.Command("schtasks.exe", "/run", "/tn", name)
	err = cmd.Run()
	cobra.CheckErr(err)

	return nil
}
