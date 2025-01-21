package ApiAuth

import (
	RestApiConfig "HosterCore/internal/app/rest_api_v2/pkg/config"
	"net/http"
)

// Check if the user is the regular REST API User, and confirms user credentials.
// Returns true if we were able to confirm both.
func CheckRestUser(r *http.Request) bool {
	user, pass, _ := r.BasicAuth()
	// fmt.Println(user, pass)
	userCheck := ""
	passCheck := ""

	// Load the REST API Config
	conf, err := RestApiConfig.GetApiConfig()
	if err != nil {
		return false
	}
	// Find the right user
	for _, v := range conf.HTTPAuth {
		if v.AdminUser {
			userCheck = v.User
			passCheck = v.Password
			break
		}
	}
	// Password cannot be empty
	if len(userCheck) < 1 || len(passCheck) < 1 || len(user) < 1 || len(pass) < 1 {
		return false
	}
	// Check user credentials
	if userCheck == user && passCheck == pass {
		return true
	}

	return false
}

// Checks if the user is an HA User, and confirms user credentials.
// Returns true if we were able to confirm both.
func CheckHaUser(r *http.Request) bool {
	user, pass, _ := r.BasicAuth()
	// fmt.Println(user, pass)
	userCheck := ""
	passCheck := ""

	// Load the REST API Config
	conf, err := RestApiConfig.GetApiConfig()
	if err != nil {
		return false
	}
	// Find the right user
	for _, v := range conf.HTTPAuth {
		if v.HaUser {
			userCheck = v.User
			passCheck = v.Password
			break
		}
	}
	// Password cannot be empty
	if len(userCheck) < 1 || len(passCheck) < 1 || len(user) < 1 || len(pass) < 1 {
		return false
	}
	// Check user credentials
	if userCheck == user && passCheck == pass {
		return true
	}

	return false
}

// Checks if the user is the Prometheus User, and confirms user credentials.
func CheckPrometheusUser(r *http.Request) bool {
	user, pass, _ := r.BasicAuth()
	// fmt.Println(user, pass)
	userCheck := ""
	passCheck := ""

	// Load the REST API Config
	conf, err := RestApiConfig.GetApiConfig()
	if err != nil {
		return false
	}
	// Find the right user
	for _, v := range conf.HTTPAuth {
		if v.PrometheusUser {
			userCheck = v.User
			passCheck = v.Password
			break
		}
	}
	// Password cannot be empty
	if len(userCheck) < 1 || len(passCheck) < 1 || len(user) < 1 || len(pass) < 1 {
		return false
	}
	// Check user credentials
	if userCheck == user && passCheck == pass {
		return true
	}

	return false
}

// Could be useful in some cases. Might delete later, after the initial testing.
func CheckAnyUser(r *http.Request) bool {
	if CheckHaUser(r) || CheckRestUser(r) || CheckPrometheusUser(r) {
		return true
	}
	return false
}
