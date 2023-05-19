package main

import (
	"bufio"
	"hoster/cmd"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

var logFileLocation string
func main() {
	// Get env vars passed from "hoster vm start"
	vmStartCommand := os.Getenv("VM_START")
	vmName := os.Getenv("VM_NAME")
	logFileLocation = os.Getenv("LOG_FILE")

	// Start the process
	parts := strings.Fields(vmStartCommand)
	for {
		logFileOutput(LOG_SUPERVISOR, "Starting the VM as a child process")
		hupCmd := exec.Command(parts[0], parts[1:]...)
		stdout, err := hupCmd.StdoutPipe()
		if err != nil {
			logFileOutput(LOG_SUPERVISOR, "Failed to create stdout pipe: " + err.Error())
			os.Exit(101)
		}
		stderr, err := hupCmd.StderrPipe()
		if err != nil {
			logFileOutput(LOG_SUPERVISOR, "Failed to create stderr pipe: " + err.Error())
			os.Exit(102)
		}

		var wg sync.WaitGroup
		wg.Add(2)

		stdoutReader := bufio.NewReader(stdout)
		go func() {
			defer wg.Done()
			readAndLogOutput(stdoutReader, LOG_SYS_OUT)
		}()

		stderrReader := bufio.NewReader(stderr)
		go func() {
			defer wg.Done()
			readAndLogOutput(stderrReader, LOG_SYS_ERR)
		}()

		done := make(chan error)
		startVmProcess(hupCmd, done)
		wg.Wait()

		processErr := <-done
		if processErr != nil {
			logFileOutput(LOG_SUPERVISOR, "VM child process ended with a non-zero exit code: " + processErr.Error())
		}

		processExitStatus, correctReturnType := processErr.(*exec.ExitError)
		if correctReturnType {
			exitCode := processExitStatus.ProcessState.ExitCode()
			if exitCode == 1 || exitCode == 2 {
				logFileOutput(LOG_SUPERVISOR, "Bhyve received a shutdown signal: " + strconv.Itoa(exitCode) + ". Shutting down...")
				logFileOutput(LOG_SUPERVISOR, "Performing network cleanup")
				cmd.NetworkCleanup(vmName, true)
				logFileOutput(LOG_SUPERVISOR, "Performing Bhyve cleanup")
				cmd.BhyvectlDestroy(vmName, true)
				logFileOutput(LOG_SUPERVISOR, "ALL CLEANUP PROCEDURES ARE DONE.")
				os.Exit(0)
			} else {
				logFileOutput(LOG_SUPERVISOR, "Bhyve returned a panic exit code: " + strconv.Itoa(exitCode))
				logFileOutput(LOG_SUPERVISOR, "Shutting down all VM related processes and performing system clean up")
				cmd.NetworkCleanup(vmName, true)
				cmd.BhyvectlDestroy(vmName, true)
				logFileOutput(LOG_SUPERVISOR, "ALL CLEANUP PROCEDURES ARE DONE.")
				os.Exit(101)
			}
		} else {
			logFileOutput(LOG_SUPERVISOR, "Bhyve received a reboot signal. Rebooting...")
			cmd.NetworkCleanup(vmName, true)
			cmd.BhyvectlDestroy(vmName, true)
			restartVmProcess(vmName)
			logFileOutput(LOG_SUPERVISOR, "ALL DONE. VM SHOULD BE UP AGAIN SHORTLY.")
			os.Exit(0)
		}
		logFileOutput(LOG_SUPERVISOR, "SOMETHING UNPREDICTED HAPPENED! THE PROCESS HAD TO EXIT!")
		cmd.NetworkCleanup(vmName, true)
		cmd.BhyvectlDestroy(vmName, true)
		os.Exit(1000)
	}
}

func readAndLogOutput(reader *bufio.Reader, name string) {
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			logFileOutput(name, err.Error())
			os.Exit(100)
		}
		line = strings.TrimSpace(line)
		if line != "" {
			logFileOutput(name, line)
		}
	}
}

func startVmProcess(cmd *exec.Cmd, done chan error) {
	err := cmd.Start()
	if err != nil {
		logFileOutput(LOG_SUPERVISOR, "Failed to start command: "+ err.Error())
		os.Exit(100)
	}
	go func() {
		done <- cmd.Wait()
	}()
}

func restartVmProcess(vmName string) {
	execPath, err := os.Executable()
	if err != nil {
		logFileOutput(LOG_SUPERVISOR, "Could not find the executable path: "+ err.Error())
		os.Exit(100)
	}
	execFile := path.Dir(execPath) + "/hoster"
	out, err := exec.Command("nohup", execFile, "vm", "start", vmName).CombinedOutput()
	if err != nil {
		removeOutputReturns := strings.ReplaceAll(string(out), "\n", "_")
		logFileOutput(LOG_SUPERVISOR, "Could not restart the VM: " + removeOutputReturns + "; " + err.Error())
		os.Exit(100)
	}
}

const LOG_SUPERVISOR = "supervisor"
const LOG_SYS_OUT = "sys_stderr"
const LOG_SYS_ERR = "sys_stdout"
func logFileOutput(msgType string, msgString string) {
	// Create or open the log file for writing
	timeNow := time.Now().Format("2006-01-02 15:04:05")
	logFile, err := os.OpenFile(logFileLocation, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0640)
	if err != nil {
		_ = exec.Command("logger", err.Error()).Run()
	}
	// log.SetOutput(logFile)
	defer logFile.Close()
	// Append the line to the file
	_, err = logFile.WriteString(timeNow + " ["+msgType+"] " + msgString + "\n")
	if err != nil {
		_ = exec.Command("logger", err.Error()).Run()
	}
}
