package cmd

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	vmSnapshotListCmd = &cobra.Command{
		Use:   "snapshot-list [vmName]",
		Short: "List VM specific snapshots",
		Long:  `List VM specific snapshot information including snapshot name, size and time taken`,
		Run: func(cmd *cobra.Command, args []string) {
			err := checkInitFile()
			if err != nil {
				log.Fatal(err.Error())
			}
			info, err := getSnapshotInfo(args[0])
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(info)
		},
	}
)

type SnapshotInfo struct {
	Name string
	Size uint64
}

func getSnapshotInfo(vmName string) (SnapshotInfo, error) {
	snapshotInfo := SnapshotInfo{}
	vmDataset, err := getVmDataset(vmName)
	if err != nil {
		return SnapshotInfo{}, err
	}

	out, err := exec.Command("zfs", "list", "-rpH", "-t", "snapshot", "-o", "name,used", vmDataset).CombinedOutput()
	fmt.Println("zfs", "list", "-rpH -t -o name,used", vmDataset)
	fmt.Println(string(out))
	if err != nil {
		return SnapshotInfo{}, err
	}

	return snapshotInfo, nil
}
