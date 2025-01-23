// Copyright 2024 Hoster Authors. All rights reserved.
// Use of this source code is governed by an Apache License 2.0
// license that can be found in the LICENSE file.

package HosterVmUtils

import (
	"HosterCore/internal/pkg/byteconversion"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

func DiskInfo(filePath string) (r DiskSize, e error) {
	reSplitSpace := regexp.MustCompile(`\s+`)

	// Total Disk Space
	out, err := exec.Command("ls", "-lh", filePath).CombinedOutput()
	if err != nil {
		e = fmt.Errorf("%s; %s", strings.TrimSpace(string(out)), err.Error())
		fmt.Println("Error TOTAL_DISK_SPACE: ", e)
		return
	}
	// Example output
	//  [0]       [1] [2]   [3]     [4] [5]  [6] [7]      [8]
	// -rw-r--r--  1 root  wheel    55G Oct  30 13:47 /tank/vm-encrypted/haDev01/disk0.img

	temp := strings.TrimSpace(string(out))
	split := reSplitSpace.Split(temp, -1)

	// r.TotalBytes, err = strconv.ParseUint(split[4], 10, 64)
	// if err != nil {
	// 	e = err
	// 	return
	// }
	// r.TotalHuman = byteconversion.BytesToHuman(r.TotalBytes)

	r.TotalBytes, err = byteconversion.HumanToBytes(strings.TrimSpace(split[4]))
	if err != nil {
		fmt.Println("Error TOTAL_BYTES: ", err)
		e = err
		return
	}
	r.TotalHuman = byteconversion.BytesToHuman(r.TotalBytes)
	// EOF Total Disk Space

	// Free Disk Space
	out, err = exec.Command("du", "-h", filePath).CombinedOutput()
	// Example output
	// [0]           [1]
	// 599M   /tank/virtio-drivers.iso
	if err != nil {
		e = fmt.Errorf("%s; %s", strings.TrimSpace(string(out)), err.Error())
		fmt.Println("Error FREE_DISK_SPACE: ", e)
		return
	}

	temp = strings.TrimSpace(string(out))
	split = reSplitSpace.Split(temp, -1)

	// r.UsedBytes, err = strconv.ParseUint(split[0], 10, 64)
	// if err != nil {
	// 	e = err
	// 	return
	// }
	// r.UsedBytes = r.UsedBytes * 1024
	r.UsedBytes, err = byteconversion.HumanToBytes(strings.TrimSpace(split[0]))
	if err != nil {
		e = err
		fmt.Printf("Error USED_BYTES: %v, %s; %s\n", e, split[0], filePath)
		return
	}

	r.UsedHuman = byteconversion.BytesToHuman(r.UsedBytes)
	// EOF Free Disk Space

	if r.UsedBytes > r.TotalBytes {
		r.TotalBytes = r.UsedBytes
		r.TotalHuman = r.UsedHuman
	}

	return
}
