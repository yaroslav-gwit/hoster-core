// Copyright 2023 Hoster Authors. All rights reserved.
// Use of this source code is governed by an Apache License 2.0
// license that can be found in the LICENSE file.

package handlers

import (
	ApiAuth "HosterCore/internal/app/rest_api_v2/pkg/auth"
	ErrorMappings "HosterCore/internal/app/rest_api_v2/pkg/error_mappings"
	JSONResponse "HosterCore/internal/app/rest_api_v2/pkg/json_response"
	HosterVm "HosterCore/internal/pkg/hoster/vm"
	"encoding/json"
	"net/http"
)

type VmStopInput struct {
	ForceCleanup bool   `json:"force_cleanup"` // Kill the VM supervisor directly (useful in the situations where you want to destroy the VM, or roll it back to a previous snapshot)
	ForceStop    bool   `json:"force_stop"`    // Send a SIGKILL instead of a graceful SIGTERM
	VmName       string `json:"vm_name"`
}

// @Tags VMs
// @Summary Stop a specific VM.
// @Description Stop a specific VM using it's name as a parameter.<br>`AUTH`: Both users are allowed.
// @Produce json
// @Security BasicAuth
// @Success 200 {object} SwaggerSuccess
// @Failure 500 {object} SwaggerError
// @Param Input body VmStopInput true "Request payload"
// @Router /vm/stop [post]
func VmStop(w http.ResponseWriter, r *http.Request) {
	if !ApiAuth.CheckAnyUser(r) {
		user, pass, _ := r.BasicAuth()
		UnauthenticatedResponse(w, user, pass)
		return
	}

	input := VmStopInput{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&input)
	if err != nil {
		ReportError(w, http.StatusInternalServerError, ErrorMappings.CouldNotParseYourInput.String())
		return
	}

	err = HosterVm.Stop(input.VmName, false, false)
	if err != nil {
		ReportError(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload, _ := JSONResponse.GenerateJson(w, "message", "success")
	SetStatusCode(w, http.StatusOK)
	w.Write(payload)
}
