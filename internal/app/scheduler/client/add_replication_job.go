// Copyright 2024 Hoster Authors. All rights reserved.
// Use of this source code is governed by an Apache License 2.0
// license that can be found in the LICENSE file.

package SchedulerClient

import (
	SchedulerUtils "HosterCore/internal/app/scheduler/utils"
	HosterJailUtils "HosterCore/internal/pkg/hoster/jail/utils"
	HosterVmUtils "HosterCore/internal/pkg/hoster/vm/utils"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
)

func AddReplicationJob(vmName string, endpoint string, key string, speedLimit int) error {
	c, err := net.Dial("unix", SchedulerUtils.SockAddr)
	if err != nil {
		return err
	}

	defer c.Close()

	job := SchedulerUtils.Job{}
	job.JobType = SchedulerUtils.JOB_TYPE_REPLICATION
	job.Replication.ResName = vmName
	job.Replication.SshEndpoint = endpoint
	job.Replication.SshKey = key
	job.Replication.SpeedLimit = speedLimit

	jsonJob, err := json.Marshal(job)
	if err != nil {
		return err
	}

	_, err = c.Write(jsonJob)
	if err != nil {
		return err
	}

	return nil
}

type RemoteDs struct {
	Name       string // normal ZFS name, aka zroot/vm-encrypted/test-vm-1, or zroot/vm-encrypted/test-vm-1@clone_test-vm-100_2023-11-29_19-55-58.222
	MountPoint string // ZFS mountpoint, e.g. /zroot/vm-encrypted/test-vm-1
}

func Replicate(job SchedulerUtils.ReplicationJob) error {
	localDs := ""
	if len(job.ResName) < 1 {
		return fmt.Errorf("resource name cannot be empty")
	}

	vms, err := HosterVmUtils.ListAllSimple()
	if err != nil {
		return err
	}
	jails, err := HosterJailUtils.ListAllSimple()
	if err != nil {
		return err
	}

	for _, v := range vms {
		if v.VmName == job.ResName {
			localDs = v.DsName + "/" + v.VmName
		}
	}
	if len(localDs) < 1 {
		for _, v := range jails {
			if v.JailName == job.ResName {
				localDs = v.DsName + "/" + v.JailName
			}
		}
	}
	if len(localDs) < 1 {
		return fmt.Errorf("could not find resource specified")
	}

	out, err := exec.Command("ssh", "-oBatchMode=yes", "-i", job.SshKey, fmt.Sprintf("-p%d", job.SshPort), job.SshEndpoint, "zfs", "list", "-t", "all", "-o", "name,mountpoint").CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not a list of remote ZFS snapshots: %s; %s", strings.TrimSpace(string(out)), err.Error())
	}

	reSplitSpace := regexp.MustCompile(`\s+`)
	remoteDs := []RemoteDs{}
	for i, v := range strings.Split(string(out), "\n") {
		if i == 0 {
			continue
		}
		if len(strings.TrimSpace(v)) < 1 {
			continue
		}

		split := reSplitSpace.Split(v, -1)
		if split[0] == localDs || strings.Contains(split[0], localDs+"@") {
			if len(split) > 1 {
				remoteDs = append(remoteDs, RemoteDs{Name: strings.TrimSpace(split[0]), MountPoint: strings.TrimSpace(split[1])})
			} else {
				remoteDs = append(remoteDs, RemoteDs{Name: strings.TrimSpace(split[0])})
			}
		}
	}

	jsonOut, err := json.MarshalIndent(remoteDs, "", "   ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonOut))
	return nil
}
