package handlers

import (
	ApiAuth "HosterCore/internal/app/rest_api_v2/pkg/auth"
	SchedulerClient "HosterCore/internal/app/scheduler/client"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
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

type CronJob struct {
	Enabled bool   `json:"enabled"`
	Time    string `json:"time"`
	User    string `json:"user"`
	Command string `json:"command"`
	Comment string `json:"comment"`
}

type CronFile struct {
	FileName      string    `json:"file_name"`
	CronVariables []string  `json:"cron_variables"`
	CronJobs      []CronJob `json:"cron_jobs"`
}

// @Tags Scheduler
// @Summary Get the list of scheduled cron jobs.
// @Description Get the list of scheduled cron jobs.<br>`AUTH`: Only REST user is allowed.
// @Produce json
// @Security BasicAuth
// @Success 200 {object} []CronFile{}
// @Failure 500 {object} SwaggerError
// @Router /scheduler/cron [get]
func SchedulerGetCron(w http.ResponseWriter, r *http.Request) {
	if !ApiAuth.CheckRestUser(r) {
		user, pass, _ := r.BasicAuth()
		UnauthenticatedResponse(w, user, pass)
		return
	}

	out, err := exec.Command("ls", "-1", "/etc/cron.d/").CombinedOutput()
	if err != nil {
		errVal := fmt.Sprintf("%s; %s", strings.TrimSpace(string(out)), err.Error())
		ReportError(w, http.StatusInternalServerError, errVal)
		return
	}

	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	cronFiles := []CronFile{}

	for _, v := range files {
		if strings.HasPrefix(v, "hoster_") {
			path := "/etc/cron.d/" + v
			cronFile, err := parseCronJob(path)
			if err != nil {
				errVal := fmt.Sprintf("%s; %s", strings.TrimSpace(string(out)), err.Error())
				log.Debugf("could not parse %s: %s", path, errVal)
			}

			cronFiles = append(cronFiles, cronFile)
		}
	}

	payload, err := json.Marshal(cronFiles)
	if err != nil {
		ReportError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Add("Content-Type", "application/json")

	SetStatusCode(w, http.StatusOK)
	w.Write(payload)
}

func parseCronJob(filePath string) (r CronFile, e error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		e = err
		return
	}

	fNameSplit := strings.Split(filePath, "/")
	r.FileName = fNameSplit[len(fNameSplit)-1]

	reSplitSpace := regexp.MustCompile(`\s+`)
	reMatchVariable := regexp.MustCompile(`^([A-Z_]+)=`)
	reMatchJobComment := regexp.MustCompile(`!('[^']*'|"[^"]*")|(\s+#.*$)`)

	variables := []string{}
	for _, v := range strings.Split(string(file), "\n") {
		v = strings.TrimSpace(v)

		if strings.HasPrefix(v, "#") {
			if strings.HasPrefix(v, "#DISABLED") || strings.HasPrefix(v, "# DISABLED") {
				_ = 0
			} else {
				continue
			}
		}

		if reMatchVariable.MatchString(v) {
			variables = append(variables, v)
			continue
		}

		job := CronJob{}
		if strings.HasPrefix(v, "#DISABLED") || strings.HasPrefix(v, "# DISABLED") {
			job.Enabled = false
			v = strings.TrimPrefix(v, "#DISABLED")
			v = strings.TrimPrefix(v, "# DISABLED")
			v = strings.TrimSpace(v)
		}

		split := reSplitSpace.Split(v, -1)
		if split[0] == "@hourly" || split[0] == "@daily" || split[0] == "@weekly" || split[0] == "@monthly" || split[0] == "@yearly" || split[0] == "@reboot" {
			job.Time = split[0]
			job.User = split[1]

			job.Comment = reMatchJobComment.FindString(v)
			job.Comment = strings.TrimSpace(job.Comment)

			job.Command = strings.Join(split[2:], " ")
			job.Command = reMatchJobComment.ReplaceAllString(job.Command, "")
			job.Command = strings.TrimSpace(job.Command)
		} else {
			job.Time = strings.Join(split[:5], " ")
			job.User = split[5]
			job.Command = strings.Join(split[6:], " ")
		}

		r.CronJobs = append(r.CronJobs, job)
	}

	r.CronVariables = variables
	return
}
