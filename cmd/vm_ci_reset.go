package cmd

import (
	"bufio"
	"errors"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var (
	newVmName          string
	ciResetNetworkName string

	vmCiResetCmd = &cobra.Command{
		Use:   "cireset",
		Short: "Reset VM's passwords, ssh keys, and network config (useful after VM migration)",
		Long:  `Reset VM's passwords, ssh keys, and network config (useful after VM migration)`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := checkInitFile()
			if err != nil {
				log.Fatal(err.Error())
			}

			err = ciReset(args[0], newVmName)
			if err != nil {
				log.Fatal("Fatal error:", err)
			}
		},
	}
)

func ciReset(oldVmName string, newVmName string) error {
	if !slices.Contains(getAllVms(), oldVmName) {
		return errors.New("vm doesn't exist")
	}
	if vmLiveCheck(oldVmName) {
		return errors.New("vm has to be stopped")
	}

	// Initialize values
	c := ConfigOutputStruct{}
	vmConfigVar := vmConfig(oldVmName)
	var err error

	// Collect the required information
	c.RootPassword = generateRandomPassword(33, true, true)
	if err != nil {
		return errors.New("could not generate random password for root user: " + err.Error())
	}

	c.GwitsuperPassword = generateRandomPassword(33, true, true)
	if err != nil {
		return errors.New("could not generate random password for gwitsuper user: " + err.Error())
	}

	c.InstanceId = generateRandomPassword(5, false, true)
	if err != nil {
		return errors.New("could not generate random instance id: " + err.Error())
	}

	if len(newVmName) > 0 {
		c.VmName = newVmName
	} else {
		c.VmName = oldVmName
	}

	c.MacAddress, err = generateRandomMacAddress()
	if err != nil {
		return errors.New("could not generate vm name: " + err.Error())
	}

	c.IpAddress, err = generateNewIp(ciResetNetworkName)
	if err != nil {
		return errors.New("could not generate the IP")
	}

	networkInfo, err := networkInfo()
	if err != nil {
		return errors.New("could not read the network config")
	}
	if len(ciResetNetworkName) < 1 {
		c.NetworkName = networkInfo[0].Name
		c.Subnet = networkInfo[0].Subnet
		c.NakedSubnet = strings.Split(networkInfo[0].Subnet, "/")[1]
		c.Gateway = networkInfo[0].Gateway
		c.NetworkComment = networkInfo[0].Comment
	} else {
		for _, v := range networkInfo {
			if ciResetNetworkName == v.Name {
				c.NetworkName = v.Name
				c.Subnet = v.Subnet
				c.NakedSubnet = strings.Split(v.Subnet, "/")[1]
				c.Gateway = v.Gateway
				c.NetworkComment = v.Comment
			}
		}
		if len(c.NetworkName) < 1 {
			return errors.New("network name supplied doesn't exist")
		}
	}

	c.Cpus = vmConfigVar.CPUCores
	c.Ram = vmConfigVar.Memory
	c.LiveStatus = vmConfigVar.LiveStatus
	c.OsType = vmConfigVar.OsType
	c.OsComment = vmConfigVar.OsComment
	c.ParentHost = GetHostName()

	c.VncPort = generateRandomVncPort()
	c.VncPassword = generateRandomPassword(8, true, true)
	if err != nil {
		return errors.New("could not generate vnc port: " + err.Error())
	}

	c.SshKeys, err = getSystemSshKeys()
	if err != nil {
		return errors.New("could not get ssh keys: " + err.Error())
	}

	oldDsName, err := getVmDataset(oldVmName)
	if err != nil {
		return errors.New(err.Error())
	}

	_ = os.Remove("/" + oldDsName + "/seed.iso")
	_ = os.RemoveAll("/" + oldDsName + "/cloud-init-files")

	// Generate template ciUserDataTemplate
	tmpl, err := template.New("ciUserDataTemplate").Parse(ciUserDataTemplate)
	if err != nil {
		return errors.New("could not generate ciUserDataTemplate: " + err.Error())
	}

	var ciUserData strings.Builder
	if err := tmpl.Execute(&ciUserData, c); err != nil {
		return errors.New("could not generate ciUserDataTemplate: " + err.Error())
	}

	// Generate template ciNetworkConfigTemplate
	tmpl, err = template.New("ciNetworkConfigTemplate").Parse(ciNetworkConfigTemplate)
	if err != nil {
		return errors.New("could not generate ciNetworkConfigTemplate: " + err.Error())
	}

	var ciNetworkConfig strings.Builder
	if err := tmpl.Execute(&ciNetworkConfig, c); err != nil {
		return errors.New("could not generate ciNetworkConfigTemplate: " + err.Error())
	}

	// Generate template ciMetaDataTemplate
	tmpl, err = template.New("ciMetaDataTemplate").Parse(ciMetaDataTemplate)
	if err != nil {
		return errors.New("could not generate ciMetaDataTemplate: " + err.Error())
	}

	var ciMetaData strings.Builder
	if err := tmpl.Execute(&ciMetaData, c); err != nil {
		return errors.New("could not generate ciMetaDataTemplate: " + err.Error())
	}

	// var vmName string
	var newDsName string
	if len(newVmName) > 0 {
		reVmNameReplace := regexp.MustCompile(`/` + oldVmName + `$`)
		newDsName = reVmNameReplace.ReplaceAllString(oldDsName, "/"+newVmName)
		if err := zfsDsRename(oldDsName, newDsName); err != nil {
			return errors.New(err.Error())
		}
		// vmName = newVmName
	} else {
		// vmName = oldVmName
		newDsName = oldDsName
	}

	// Write config files
	newVmFolder := "/" + newDsName

	vmConfigVar.Networks[0].NetworkBridge = c.NetworkName
	vmConfigVar.Networks[0].NetworkMac = c.MacAddress
	vmConfigVar.Networks[0].IPAddress = c.IpAddress
	vmConfigVar.Networks[0].Comment = c.NetworkComment
	vmConfigVar.ParentHost = c.ParentHost
	vmConfigVar.VncPort = c.VncPort
	vmConfigVar.VncPassword = c.VncPassword
	vmConfigVar.VmSshKeys = c.SshKeys

	vmConfigFileLocation := newVmFolder + "/vm_config.json"
	err = vmConfigFileWriter(vmConfigVar, vmConfigFileLocation)
	if err != nil {
		return err
	}

	// Create cloud init folder
	if _, err := os.Stat(newVmFolder + "/cloud-init-files"); os.IsNotExist(err) {
		err = os.Mkdir(newVmFolder+"/cloud-init-files", 0750)
		if err != nil {
			return err
		}
	}

	// Open /cloud-init-files/user-data for writing
	ciUserDataFileLocation, err := os.Create(newVmFolder + "/cloud-init-files/user-data")
	if err != nil {
		return errors.New(err.Error())
	}
	defer ciUserDataFileLocation.Close()
	// Create a new writer
	writer := bufio.NewWriter(ciUserDataFileLocation)
	// Write a string to the file
	str := ciUserData.String()
	_, err = writer.WriteString(str)
	if err != nil {
		return errors.New(err.Error())
	}
	// Flush the writer to ensure all data has been written to the file
	err = writer.Flush()
	if err != nil {
		return errors.New(err.Error())
	}

	// Open /cloud-init-files/network-config for writing
	ciNetworkFileLocation, err := os.Create(newVmFolder + "/cloud-init-files/network-config")
	if err != nil {
		return errors.New(err.Error())
	}
	defer ciNetworkFileLocation.Close()
	// Create a new writer
	writer = bufio.NewWriter(ciNetworkFileLocation)
	// Write a string to the file
	str = ciNetworkConfig.String()
	_, err = writer.WriteString(str)
	if err != nil {
		return errors.New(err.Error())
	}
	// Flush the writer to ensure all data has been written to the file
	err = writer.Flush()
	if err != nil {
		return errors.New(err.Error())
	}

	// Open /cloud-init-files/meta-data for writing
	ciMetaDataFileLocation, err := os.Create(newVmFolder + "/cloud-init-files/meta-data")
	if err != nil {
		return errors.New(err.Error())
	}
	defer ciMetaDataFileLocation.Close()
	// Create a new writer
	writer = bufio.NewWriter(ciMetaDataFileLocation)
	// Write a string to the file
	str = ciMetaData.String()
	_, err = writer.WriteString(str)
	if err != nil {
		return errors.New(err.Error())
	}
	// Flush the writer to ensure all data has been written to the file
	err = writer.Flush()
	if err != nil {
		return errors.New(err.Error())
	}

	err = createCiIso(newVmFolder)
	if err != nil {
		return errors.New(err.Error())
	}

	err = reloadDnsServer()
	if err != nil {
		return err
	}

	return nil
}

func zfsDsRename(oldDsName, newDsName string) error {
	err := exec.Command("zfs", "rename", oldDsName, newDsName).Run()
	if err != nil {
		return errors.New("could not execute zfs snapshot: " + err.Error())
	}
	return nil
}
