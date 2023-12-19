package cmd

import (
	"HosterCore/emojlog"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	jailStartAllCmd = &cobra.Command{
		Use:   "start-all",
		Short: "Start all available Jails on this system",
		Long:  `Start all available Jails on this system.`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			checkInitFile()

			err := startAllJails(true)
			if err != nil {
				emojlog.PrintLogMessage(err.Error(), emojlog.Error)
				os.Exit(1)
			}
		},
	}
)

func startAllJails(consoleLogOutput bool) error {
	jailList, err := GetAllJailsList()
	if err != nil {
		return err
	}

	startId := 0
	for _, v := range jailList {
		jailConfig, err := GetJailConfig(v, true)
		if err != nil {
			return err
		}
		jailOnline, err := checkJailOnline(jailConfig)
		if err != nil {
			return err
		}
		if jailOnline {
			continue
		}

		// Print out the output splitter
		if startId == 0 {
			_ = 0
		} else {
			fmt.Println("  ───────────")
		}

		if startId != 0 {
			time.Sleep(3 * time.Second)
		}

		err = jailStart(v, consoleLogOutput)
		if err != nil {
			return err
		}

		startId += 1
	}

	return nil
}
