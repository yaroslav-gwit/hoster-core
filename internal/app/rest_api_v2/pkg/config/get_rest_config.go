package RestApiConfig

import (
	FileExists "HosterCore/internal/pkg/file_exists"
	HosterLocations "HosterCore/internal/pkg/hoster/locations"
	"encoding/json"
	"errors"
	"os"
)

type RestApiConfig struct {
	BindToAddress string `json:"bind"`      // can be empty, 0.0.0.0 used by default
	Port          int    `json:"port"`      // port to bind the HTTP server to
	Protocol      string `json:"protocol"`  // http or https -> not implemented yet, will require another parameter: key_location
	HaMode        bool   `json:"ha_mode"`   // whether to start the API server in an HA cluster mode
	HaDebug       bool   `json:"ha_debug"`  // ha_debug allows you to test the HA Mode, because instead of applying the real actions, ha_debug will only log them instead
	LogLevel      string `json:"log_level"` // DEBUG, INFO, WARN, or ERROR
	HTTPAuth      []struct {
		User           string `json:"user"`            // user name for the basic HTTP auth
		Password       string `json:"password"`        // password for the basic HTTP auth
		HaUser         bool   `json:"ha_user"`         // HA User has access to a different set of routes than the regular REST API user, and vise versa. Has been implemented to limit per-user API exposure, aka normal user is not authorized to call HA related routes.
		PrometheusUser bool   `json:"prometheus_user"` // Prometheus User has access to the Prometheus metrics endpoint
		AdminUser      bool   `json:"admin_user"`      // Admin User has access to the Admin API routes
	} `json:"http_auth"`
}

const confFileName = "restapi_config.json"

// A function, that loops through the list of possible
// config locations and picks up the first one available.
func GetApiConfigLocation() (r string, e error) {
	for _, v := range HosterLocations.GetConfigFolders() {
		configLocation := v + "/" + confFileName
		if FileExists.CheckUsingOsStat(configLocation) {
			r = configLocation
			return
		}
	}

	e = errors.New("could not find the config file: " + confFileName)
	return
}

// Parses the host_config.json, and returns the underlying struct or an error
func GetApiConfig() (r RestApiConfig, e error) {
	apiConfigFile, err := GetApiConfigLocation()
	if err != nil {
		e = err
		return
	}

	data, err := os.ReadFile(apiConfigFile)
	if err != nil {
		e = err
		return
	}

	err = json.Unmarshal(data, &r)
	if err != nil {
		e = err
		return
	}

	if len(r.LogLevel) < 1 {
		r.LogLevel = "DEBUG"
	}

	return
}
