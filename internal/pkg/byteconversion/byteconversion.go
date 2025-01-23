// Copyright 2023 Hoster Authors. All rights reserved.
// Use of this source code is governed by an Apache License 2.0
// license that can be found in the LICENSE file.

package byteconversion

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func BytesToHuman(bytes uint64) string {
	// SET TO KB
	var firstIteration = bytes / 1024
	var iterationTitle = "K"
	// SET TO MB
	if firstIteration > 1024 {
		firstIteration = firstIteration / 1024
		iterationTitle = "M"
	}
	// SET TO GB
	var firstIterationFloat = 0.0
	if firstIteration > 1024 {
		firstIterationFloat = float64(firstIteration) / 1024.0
		iterationTitle = "G"
	}
	// FORMAT THE OUTPUT
	var r string
	if firstIterationFloat > 0.0 {
		r = fmt.Sprintf("%.2f%s", firstIterationFloat, iterationTitle)
	} else {
		r = fmt.Sprintf("%d%s", firstIteration, iterationTitle)
	}

	return r
}

func HumanToBytes(sizeString string) (r uint64, e error) {
	sizeString = strings.TrimSpace(strings.ToUpper(sizeString))
	var multiplier uint64 = 1

	// Double characters conversion
	switch {
	case strings.HasSuffix(sizeString, "TB"):
		multiplier = 1 << 40
		sizeString = strings.TrimSuffix(sizeString, "TB")
	case strings.HasSuffix(sizeString, "GB"):
		multiplier = 1 << 30
		sizeString = strings.TrimSuffix(sizeString, "GB")
	case strings.HasSuffix(sizeString, "MB"):
		multiplier = 1 << 20
		sizeString = strings.TrimSuffix(sizeString, "MB")
	case strings.HasSuffix(sizeString, "KB"):
		multiplier = 1 << 10
		sizeString = strings.TrimSuffix(sizeString, "KB")
	case strings.HasSuffix(sizeString, "B"):
		sizeString = strings.TrimSuffix(sizeString, "B")
	}

	// Single character conversion
	switch {
	case strings.HasSuffix(sizeString, "T"):
		multiplier = 1 << 40
		sizeString = strings.TrimSuffix(sizeString, "T")
	case strings.HasSuffix(sizeString, "G"):
		multiplier = 1 << 30
		sizeString = strings.TrimSuffix(sizeString, "G")
	case strings.HasSuffix(sizeString, "M"):
		multiplier = 1 << 20
		sizeString = strings.TrimSuffix(sizeString, "M")
	case strings.HasSuffix(sizeString, "K"):
		multiplier = 1 << 10
		sizeString = strings.TrimSuffix(sizeString, "K")
	}

	sizeFloat, err := strconv.ParseFloat(sizeString, 64)
	if err != nil {
		e = fmt.Errorf("invalid size format: %s", err.Error())
		return
	}

	// Parse Uint64 from float64
	sizeFloat = sizeFloat * float64(multiplier)
	sizeFloat = math.Round(sizeFloat)
	size := uint64(sizeFloat)

	// size, err := strconv.ParseUint(sizeString, 10, 64)
	// if err != nil {
	// 	e = fmt.Errorf("invalid size format: %s", err.Error())
	// 	return
	// }

	// r = size * multiplier
	r = size
	return
}
