package handlers

import (
	ApiAuth "HosterCore/internal/app/rest_api_v2/pkg/auth"
	"HosterCore/internal/pkg/byteconversion"
	HosterHost "HosterCore/internal/pkg/hoster/host"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// @Tags Datasets
// @Summary Get active dataset list.
// @Description Get active dataset list.<br>`AUTH`: Only REST user is allowed.
// @Produce json
// @Security BasicAuth
// @Success 200 {object} []DatasetInfo
// @Failure 500 {object} SwaggerError
// @Router /dataset/all [get]
func DatasetList(w http.ResponseWriter, r *http.Request) {
	if !ApiAuth.CheckRestUser(r) {
		user, pass, _ := r.BasicAuth()
		UnauthenticatedResponse(w, user, pass)
		return
	}

	hostConf, err := HosterHost.GetHostConfig()
	if err != nil {
		ReportError(w, http.StatusInternalServerError, err.Error())
		return
	}

	info := []DatasetInfo{}
	for _, v := range hostConf.ActiveZfsDatasets {
		dsInfo, err := getDsInfo(v)
		if err != nil {
			continue
		}

		info = append(info, dsInfo)
	}

	payload, err := json.Marshal(info)
	if err != nil {
		ReportError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SetStatusCode(w, http.StatusOK)
	w.Write(payload)
}

type DatasetInfo struct {
	Encrypted      bool   `json:"encrypted"`
	Mounted        bool   `json:"mounted"`
	Used           uint64 `json:"used"`
	Available      uint64 `json:"available"`
	Total          uint64 `json:"total"`
	UsedHuman      string `json:"used_human"`
	AvailableHuman string `json:"available_human"`
	TotalHuman     string `json:"total_human"`
	Name           string `json:"name"`
}

func getDsInfo(dsName string) (r DatasetInfo, e error) {
	out, err := exec.Command("zfs", "list", "-p", "-o", "name,used,available,mounted,encryption", dsName).CombinedOutput()
	if err != nil {
		e = fmt.Errorf("%s; %s", strings.TrimSpace(string(out)), err.Error())
		return
	}

	// Encrypted example:
	// zfs list -p -o name,used,available,mounted,encryption tank/vm-encrypted
	//
	// NAME                USED          AVAIL        MOUNTED  ENCRYPTION
	// [0]                 [1]           [2]          [3]           [4]
	// tank/vm-encrypted  36383670272  1329563111424  yes      aes-256-gcm

	// Unencrypted example:
	// zfs list -p -o name,used,available,mounted,encryption tank/vm-unencrypted
	//
	// NAME                  USED       AVAIL       MOUNTED  ENCRYPTION
	// [0]                  [1]         [2]          [3]      [4]
	// tank/vm-unencrypted  98304   1329563111424    yes      off

	reSpace := regexp.MustCompile(`\s+`)
	realValues := strings.Split(string(out), "\n")[1]
	split := reSpace.Split(realValues, -1)

	used, err := strconv.ParseUint(split[1], 10, 64)
	if err != nil {
		e = err
		return
	}
	usedHuman := byteconversion.BytesToHuman(used)

	available, err := strconv.ParseUint(split[2], 10, 64)
	if err != nil {
		e = err
		return
	}
	availableHuman := byteconversion.BytesToHuman(available)

	r.Name = split[0]
	r.Available = available
	r.AvailableHuman = availableHuman

	r.Used = used
	r.UsedHuman = usedHuman

	r.Total = used + available
	r.TotalHuman = byteconversion.BytesToHuman(r.Total)

	r.Mounted = split[3] == "yes"
	r.Encrypted = split[4] != "off"

	return
}
