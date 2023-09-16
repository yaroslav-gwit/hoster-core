package main

import (
	"hoster/cmd"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	_ = exec.Command("logger", "-t", "HOSTER_HA_WATCHDOG", "DEBUG: ha_watchdog service start-up").Run()

	var debugMode bool
	debugModeEnv := os.Getenv("REST_API_HA_DEBUG")
	if len(debugModeEnv) > 0 {
		debugMode = true
	}

	if debugMode {
		_ = exec.Command("logger", "-t", "HOSTER_HA_REST", "DEBUG: ha_watchdog started in DEBUG mode").Run()
	} else {
		_ = exec.Command("logger", "-t", "HOSTER_HA_REST", "PROD: ha_watchdog started in PRODUCTION mode").Run()
	}

	timesFailed := 0
	timesFailedMax := 2
	lastReachOut := time.Now().Unix()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for sig := range signals {
			if sig == syscall.SIGHUP {
				lastReachOut = time.Now().Unix()
			}
			if sig == syscall.SIGTERM || sig == syscall.SIGINT {
				_ = exec.Command("logger", "-t", "HOSTER_HA_WATCHDOG", "INFO: received SIGTERM, exiting").Run()
				os.Exit(0)
			}
		}
	}()

	for {
		time.Sleep(time.Second * 5)

		if timesFailed >= timesFailedMax {
			if debugMode {
				_ = exec.Command("logger", "-t", "HOSTER_HA_WATCHDOG", "DEBUG: HOSTER_HA_REST process has failed, rebooting the system now").Run()
				os.Exit(1)
			} else {
				_ = exec.Command("logger", "-t", "HOSTER_HA_WATCHDOG", "FATAL: HOSTER_HA_REST process has failed, rebooting the system now").Run()
				cmd.LockAllVms()
				_ = exec.Command("reboot").Run()
			}
		}

		if time.Now().Unix() > lastReachOut+5 {
			// _ = exec.Command("logger", "-t", "HOSTER_HA_WATCHDOG", "ping failed, previous alive timestamp: "+strconv.Itoa(int(lastReachOut))).Run()
			timesFailed += 1
			// _ = exec.Command("logger", "-t", "HOSTER_HA_WATCHDOG", "pings missed so far: "+strconv.Itoa(timesFailed)+"; will terminate the system at: "+strconv.Itoa(timesFailedMax)).Run()
		} else {
			timesFailed = 0
		}
	}
}
