package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var (
	jailStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start a specific Jail",
		Long:  `Start a specific Jail using it's name`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := checkInitFile()
			if err != nil {
				log.Fatal(err.Error())
			}
			// cmd.Help()

			err = jailStart(args[0])
			if err != nil {
				log.Fatal(err.Error())
			}
		},
	}
)

func jailStart(jailName string) error {
	jailConfig, err := getJailConfig(jailName)
	if err != nil {
		return err
	}

	t, err := template.New("jailRunningConfigPartialTemplate").Parse(jailRunningConfigPartialTemplate)
	if err != nil {
		return err
	}

	var jailConfigBuffer bytes.Buffer
	err = t.Execute(&jailConfigBuffer, jailConfig)
	if err != nil {
		return err
	}

	var jailConfigString = jailConfigBuffer.String()

	var additionalConfig []byte
	if FileExists(jailConfig.JailFolder + jailConfig.ConfigFileAppend) {
		additionalConfig, err = os.ReadFile(jailConfig.JailFolder + jailConfig.ConfigFileAppend)
		if err != nil {
			return err
		}
	}

	if len(additionalConfig) > 0 {
		additionalConfigSplit := strings.Split(string(additionalConfig), "\n")
		for _, v := range additionalConfigSplit {
			if len(v) > 0 {
				v = strings.TrimSpace(v)
				jailConfigString = jailConfigString + "    " + v + "\n"
			}
		}
		jailConfigString = jailConfigString + "}"
	} else {
		jailConfigString = jailConfigString + "\n}"
	}

	_ = os.Remove(jailConfig.JailFolder + "jail_temp_runtime.conf")
	err = os.WriteFile(jailConfig.JailFolder+"jail_temp_runtime.conf", []byte(jailConfigString), 0644)
	if err != nil {
		return err
	}

	jailCommandOutput, err := exec.Command("jail", "-c", "-f", jailConfig.JailFolder+"jail_temp_runtime.conf").CombinedOutput()
	if err != nil {
		errorValue := "FATAL: " + strings.TrimSpace(string(jailCommandOutput)) + "; " + err.Error()
		return errors.New(errorValue)
	}

	// fmt.Println(jailConfigString)
	return nil
}

const jailRunningConfigPartialTemplate = `# Running Jail config generated by Hoster
{{ .JailName }} {
    host.hostname = {{ .JailHostname }};
    ip4.addr = "vm-{{ .Network }}|{{ .IPAddress }}/{{ .Netmask }}";
    path = "{{ .JailRootPath }}";
    exec.clean;
    exec.start = "{{ .StartupScript }}";
    exec.stop = "{{ .ShutdownScript }}";

    # Log Jail startup and shutdown
    exec.prestart = "logger HOSTER_JAILS starting the Jail: {{ .JailName }}";
    exec.poststart = "logger HOSTER_JAILS the Jail has been started: {{ .JailName }}";
    exec.prestop = "logger HOSTER_JAILS stopping the Jail: {{ .JailName }}";
    exec.poststop = "logger HOSTER_JAILS the Jail has been stopped: {{ .JailName }}";

    # Apply Jail resource limits
    exec.poststart += "rctl -a jail:{{ .JailName }}:vmemoryuse:deny={{ .RAMLimit }}";
    exec.poststart += "rctl -a jail:{{ .JailName }}:pcpu:deny={{ .CPULimitPercent }}";
    exec.poststop += "rctl -r jail:{{ .JailName }}";

    # Additional config
`

func checkIfJailExists(jailName string) (jailExists bool) {
	datasets, err := getZfsDatasetInfo()
	if err != nil {
		return
	}

	for _, v := range datasets {
		if FileExists(v.MountPoint + "/" + jailName + "/jail_config.json") {
			return true
		}
	}

	return
}

func getJailConfig(jailName string) (jailConfig JailConfigFileStruct, configError error) {
	if !checkIfJailExists(jailName) {
		configError = errors.New("jail doesn't exist")
		return
	}

	datasets, err := getZfsDatasetInfo()
	if err != nil {
		configError = err
		return
	}

	configFile := ""
	for _, v := range datasets {
		if FileExists(v.MountPoint + "/" + jailName + "/jail_config.json") {
			configFile = v.MountPoint + "/" + jailName + "/jail_config.json"
			jailConfig.JailFolder = v.MountPoint + "/" + jailName + "/"
			jailConfig.JailRootPath = v.MountPoint + "/" + jailName + "/root_folder"
		}
	}

	configFileRead, err := os.ReadFile(configFile)
	if err != nil {
		configError = err
		return
	}

	unmarshalErr := json.Unmarshal(configFileRead, &jailConfig)
	if unmarshalErr != nil {
		configError = unmarshalErr
		return
	}

	jailConfig.JailName = jailName
	jailConfig.JailHostname = jailName + "." + GetHostName() + ".internal.lan"
	jailConfig.Netmask = "24"

	commandOutput, err := exec.Command("sysctl", "-nq", "hw.ncpu").CombinedOutput()
	if err != nil {
		fmt.Println("Error", err.Error())
	}

	numberOfCpusInt, err := strconv.Atoi(strings.TrimSpace(string(commandOutput)))
	if err != nil {
		configError = err
		return
	}

	realCpuLimit := jailConfig.CPULimitPercent * numberOfCpusInt
	jailConfig.CPULimitPercent = realCpuLimit

	return
}
