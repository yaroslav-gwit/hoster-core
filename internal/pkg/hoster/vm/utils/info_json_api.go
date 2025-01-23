// Copyright 2024 Hoster Authors. All rights reserved.
// Use of this source code is governed by an Apache License 2.0
// license that can be found in the LICENSE file.

package HosterVmUtils

import (
	FreeBSDps "HosterCore/internal/pkg/freebsd/ps"
	FreeBSDsysctls "HosterCore/internal/pkg/freebsd/sysctls"
	timeconversion "HosterCore/internal/pkg/time_conversion"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

func InfoJsonApi(vmName string) (r VmApi, e error) {
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
		if v.VmName != vmName {
			continue
		}

		conf, err := GetVmConfig(v.Mountpoint + "/" + v.VmName)
		if err != nil {
			continue
		}

		r.Name = v.VmName
		r.VmConfig = conf
		r.Encrypted = v.Encrypted
		r.Simple = v

		if slices.Contains(liveVms, v.VmName) {
			r.Running = true
			reMatch := regexp.MustCompile(`bhyve:\s+` + v.VmName + `($|\s+)`)
			for _, vv := range ps {
				if reMatch.MatchString(vv.Command) {
					r.UptimeUnix = vv.ElapsedTime
					r.Uptime = timeconversion.ProcessUptimeToHuman(vv.ElapsedTime)
					break
				}
			}
		} else {
			r.Uptime = "0s"
		}

		r.CurrentHost = hostname
		if hostname != conf.ParentHost {
			r.Backup = true
		}

		for ii, vv := range conf.Disks {
			if vv.DiskLocation == "internal" && !strings.HasPrefix(vv.DiskImage, "/") {
				diskInfo, err := DiskInfo(v.Mountpoint + "/" + v.VmName + "/" + vv.DiskImage)
				if err != nil {
					continue
				}
				r.Disks[ii].DiskSize = diskInfo
			} else if vv.DiskLocation == "external" || strings.HasPrefix(vv.DiskImage, "/") {
				diskInfo, err := DiskInfo(vv.DiskImage)
				if err != nil {
					continue
				}
				r.Disks[ii].DiskSize = diskInfo
			}
		}

		return
	}

	e = fmt.Errorf("sorry, could not find this VM")
	return
}
