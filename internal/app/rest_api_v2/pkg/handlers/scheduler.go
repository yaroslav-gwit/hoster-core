package handlers

import (
	ApiAuth "HosterCore/internal/app/rest_api_v2/pkg/auth"
	SchedulerClient "HosterCore/internal/app/scheduler/client"
	"encoding/json"
	"net/http"
)

// @Tags Scheduler
// @Summary Get the list of scheduled, active and past jobs.
// @Description Get the list of scheduled, active and past jobs.<br>`AUTH`: Only REST user is allowed.
// @Produce json
// @Security BasicAuth
// @Success 200 {object} []SchedulerUtils.Job{}
// @Failure 500 {object} SwaggerError
// @Router /scheduler/jobs [get]
func SchedulerGetJobs(w http.ResponseWriter, r *http.Request) {
	if !ApiAuth.CheckRestUser(r) {
		user, pass, _ := r.BasicAuth()
		UnauthenticatedResponse(w, user, pass)
		return
	}

	jobs, err := SchedulerClient.GetJobList()
	if err != nil {
		ReportError(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload, err := json.Marshal(jobs)
	if err != nil {
		ReportError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Add("Content-Type", "application/json")

	SetStatusCode(w, http.StatusOK)
	w.Write(payload)
}
