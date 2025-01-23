// Copyright 2024 Hoster Authors. All rights reserved.
// Use of this source code is governed by an Apache License 2.0
// license that can be found in the LICENSE file.

package HosterVmUtils

import (
	FreeBSDps "HosterCore/internal/pkg/freebsd/ps"
	FreeBSDsysctls "HosterCore/internal/pkg/freebsd/sysctls"
	timeconversion "HosterCore/internal/pkg/time_conversion"
	"regexp"
	"slices"
	"strings"
)

type DiskSize struct {
	TotalBytes uint64 `json:"total_bytes,omitempty"`
	TotalHuman string `json:"total_human,omitempty"`
	UsedBytes  uint64 `json:"used_bytes,omitempty"`
	UsedHuman  string `json:"used_human,omitempty"`
}

type VmApi struct {
	VmConfig
	Simple      VmListSimple `json:"simple"`
	Name        string       `json:"name"`
	Uptime      string       `json:"uptime"`
	UptimeUnix  int64        `json:"uptime_unix"`
	Running     bool         `json:"running"`
	Backup      bool         `json:"backup"`
	Encrypted   bool         `json:"encrypted"`
	CurrentHost string       `json:"current_host"`
	// Metrics     rctl.RctMetrics `json:"rctl_metrics,omitempty"`
}

func ListJsonApi() (r []VmApi, e error) {
	ps, err := FreeBSDps.ProcessTimes()
	if err != nil {
		e = err
		return
	}

	vms, err := ListAllSimple()
	if err != nil {
		e = err
		return
	}
	liveVms, err := GetRunningVms()
	if err != nil {
		e = err
		return
	}

	hostname, _ := FreeBSDsysctls.SysctlKernHostname()
	for _, v := range vms {
		temp := VmApi{}
		conf, err := GetVmConfig(v.Mountpoint + "/" + v.VmName)
		if err != nil {
			// fmt.Println(err)
			continue
		}

		temp.Name = v.VmName
		temp.VmConfig = conf
		temp.Encrypted = v.Encrypted
		temp.Simple = v

		if slices.Contains(liveVms, v.VmName) {
			temp.Running = true
			reMatch := regexp.MustCompile(`bhyve:\s+` + v.VmName + `($|\s+)`)
			for _, vv := range ps {
				// fmt.Println(vv.Command)
				if reMatch.MatchString(vv.Command) {
					// fmt.Println(vv.StartTime)
					temp.UptimeUnix = vv.ElapsedTime
					temp.Uptime = timeconversion.ProcessUptimeToHuman(vv.ElapsedTime)
					break
				}

				// temp.Metrics, err = rctl.MetricsProcess(vv.PID)
				// if err != nil {
				// 	temp.Metrics = rctl.RctMetrics{}
				// }
			}
		} else {
			temp.Uptime = "0s"
		}

		temp.CurrentHost = hostname
		if hostname != conf.ParentHost {
			temp.Backup = true
		}

		// for ii, vv := range conf.Disks {
		// 	diskInfo, err := DiskInfo(v.Mountpoint + "/" + v.VmName + "/" + vv.DiskImage)
		// 	if err != nil {
		// 		continue
		// 	}
		// 	temp.Disks[ii].DiskSize = diskInfo
		// }
		for ii, vv := range conf.Disks {
			if vv.DiskLocation == "internal" && !strings.HasPrefix(vv.DiskImage, "/") {
				diskInfo, err := DiskInfo(v.Mountpoint + "/" + v.VmName + "/" + vv.DiskImage)
				if err != nil {
					continue
				}
				temp.Disks[ii].DiskSize = diskInfo
			} else if vv.DiskLocation == "external" || strings.HasPrefix(vv.DiskImage, "/") {
				diskInfo, err := DiskInfo(vv.DiskImage)
				if err != nil {
					continue
				}
				temp.Disks[ii].DiskSize = diskInfo
			}
		}

		r = append(r, temp)
	}

	return
}
