//go:build freebsd
// +build freebsd

package main

import (
	"HosterCore/internal/app/rest_api_v2/pkg/handlers"
	HandlersHA "HosterCore/internal/app/rest_api_v2/pkg/handlers_ha"
	MiddlewareLogging "HosterCore/internal/app/rest_api_v2/pkg/middleware/logging"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	_ "github.com/swaggo/http-swagger/example/gorilla/docs" // docs is generated by Swag CLI, you have to import it.
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

var log *MiddlewareLogging.Log
var version = "" // This is set by the build script

// @title Hoster Node REST API Docs
// @version 2.0
// @securityDefinitions.basic BasicAuth
// @description `NOTE!` This REST API HTTP endpoint is located directly on the `Hoster` node.<br><br>The API should ideally be integrated into another system (e.g. a user-accessible back-end server), and not interacted with directly.<br><br>Please, take an extra care with the things you execute here, because some of them may be disruptive or non-revertible (e.g. vm destroy, snapshot rollback, host reboot, etc).
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /api/v2
func main() {
	// Print the version and exit
	args := os.Args
	if len(args) > 1 {
		res := os.Args[1]
		if res == "version" || res == "v" || res == "--version" || res == "-v" {
			fmt.Println(version)
			return
		}
	}

	r := mux.NewRouter()
	// log = MiddlewareLogging.Configure(logrus.DebugLevel)

	// Middleware -> Logging
	log = MiddlewareLogging.Configure(logrus.DebugLevel)
	handlers.SetLogConfig(log)
	r.Use(log.LogResponses)

	// Health checks
	// r.HandleFunc("/api/v2/health", handlers.HealthCheck).Methods("GET")
	r.HandleFunc("/api/v2/health", handlers.HealthCheck).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/health/auth/ha", handlers.HealthCheckHaAuth).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/health/auth/any", handlers.HealthCheckAnyAuth).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/health/auth/regular", handlers.HealthCheckRegularAuth).Methods(http.MethodGet)
	// Host
	r.HandleFunc("/api/v2/host/info", handlers.HostInfo).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/host/readme", handlers.GetHostReadme).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/host/settings", handlers.HostSettings).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/host/settings/add-tag/{tag}", handlers.PostHostTag).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/host/settings/delete-tag/{tag}", handlers.DeleteHostTag).Methods(http.MethodDelete)
	r.HandleFunc("/api/v2/host/settings/delete-tag/{tag}", handlers.DeleteHostTag).Methods(http.MethodPost) // additional POST method for the clients that do not support DELETE
	r.HandleFunc("/api/v2/host/settings/api", handlers.HostRestApiSettings).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/host/settings/dns-search-domain", handlers.PostHostSettingsDnsSearchDomain).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/host/settings/vm-templates", handlers.PostHostSettingsVmTemplateLink).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/host/settings/add-upstream-dns", handlers.PostHostSettingsAddUpstreamDns).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/host/settings/delete-upstream-dns", handlers.DeleteHostSettingsUpstreamDns).Methods(http.MethodDelete)
	r.HandleFunc("/api/v2/host/settings/delete-upstream-dns", handlers.DeleteHostSettingsUpstreamDns).Methods(http.MethodPost) // additional POST method for the clients that do not support DELETE
	r.HandleFunc("/api/v2/host/settings/ssh-auth-key", handlers.PostHostSshAuthKey).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/host/settings/add-ssh-key", handlers.PostHostSettingsSshKey).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/host/settings/delete-ssh-key", handlers.DeleteHostSettingsSshKey).Methods(http.MethodDelete)
	r.HandleFunc("/api/v2/host/settings/delete-ssh-key", handlers.DeleteHostSettingsSshKey).Methods(http.MethodPost) // additional POST method for the clients that do not support DELETE
	// Datasets
	r.HandleFunc("/api/v2/dataset/all", handlers.DatasetList).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/dataset/unlock", handlers.UnlockEncryptedDataset).Methods(http.MethodPost)
	// Networks
	r.HandleFunc("/api/v2/network/all", handlers.NetworkList).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/network/add-new-network", handlers.PostNewNetwork).Methods(http.MethodPost)
	// VMs
	r.HandleFunc("/api/v2/vm/all", handlers.VmList).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/vm/all/cache", handlers.VmListCache).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/vm/info/{vm_name}", handlers.VmInfo).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/vm/settings/{vm_name}", handlers.VmGetSettings).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/vm/settings/cpu/{vm_name}", handlers.VmPostCpuInfo).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/ram/{vm_name}", handlers.VmPostRamInfo).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/os-info/{vm_name}", handlers.VmPostOsSettings).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/vnc-resolution/{vm_name}/{resolution}", handlers.VmPostVncResolution).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/firmware/{vm_name}/{firmware}", handlers.VmPostFirmwareType).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/production/{vm_name}/{production}", handlers.VmPostProductionSetting).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/add-tag/{vm_name}", handlers.VmPostNewTag).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/delete-tag/{vm_name}", handlers.VmDeleteExistingTag).Methods(http.MethodDelete, http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/mount-iso/{vm_name}", handlers.VmPostMountIso).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/unmount-iso/{vm_name}", handlers.VmPostUnmountIso).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/disk/add-new/{vm_name}", handlers.VmPostAddNewDisk).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/disk/expand/{vm_name}", handlers.VmPostExpandDisk).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/network/add/{vm_name}", handlers.VmPostAddNewNetwork).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/settings/description/{vm_name}", handlers.VmPostDescription).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/templates", handlers.VmGetTemplates).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/cloud-init/mount-iso/{vm_name}", handlers.VmPostMountCiIso).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/cloud-init/unmount-iso/{vm_name}", handlers.VmPostUnmountCiIso).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/readme/{vm_name}", handlers.VmGetReadme).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/vm/start/{vm_name}", handlers.VmPostStart).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/start/wait-vnc/{vm_name}", handlers.VmPostStartAndWaitVnc).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/start-all/{production}", handlers.VmPostStartAll).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/stop-all/{force}", handlers.VmPostStopAll).Methods(http.MethodPost)
	// r.HandleFunc("/api/v2/vm/stop", handlers.VmStop).Methods(http.MethodPost) // Deprecated old method
	r.HandleFunc("/api/v2/vm/stop/{vm_name}", handlers.VmPostStop).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/stop/force/{vm_name}", handlers.VmPostStopForce).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/clone", handlers.VmClone).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/deploy", handlers.VmPostDeploy).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/vm/destroy/{vm_name}", handlers.VmDestroy).Methods(http.MethodDelete, http.MethodPost)
	// Jails
	r.HandleFunc("/api/v2/jail/all", handlers.JailList).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/jail/all/cache", handlers.JailListCache).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/jail/template/list", handlers.JailListTemplates).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/jail/info/{jail_name}", handlers.JailInfo).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/jail/start/{jail_name}", handlers.JailStart).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/start-all/{production}", handlers.JailPostStartAll).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/stop-all", handlers.JailPostStopAll).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/stop/{jail_name}", handlers.JailStop).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/clone", handlers.JailClone).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/destroy/{jail_name}", handlers.JailDestroy).Methods(http.MethodDelete)
	r.HandleFunc("/api/v2/jail/deploy", handlers.JailDeploy).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/readme/{jail_name}", handlers.JailGetReadme).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/jail/get/shells/{jail_name}", handlers.JailGetShells).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/jail/settings/{jail_name}", handlers.JailGetSettings).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/jail/settings/description/{jail_name}", handlers.JailPostDescription).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/settings/add-tag/{jail_name}", handlers.JailPostNewTag).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/settings/production/{jail_name}/{production}", handlers.JailPostProductionSetting).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/settings/cpu/{jail_name}/{limit}", handlers.JailPostCpuPercentageLimit).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/settings/ram/{jail_name}/{limit}", handlers.JailPostRamLimit).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/settings/dns/{jail_name}", handlers.JailPostSettingsDns).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/jail/settings/network/{jail_name}", handlers.JailPostSettingsNetwork).Methods(http.MethodPost)
	// Snapshots
	r.HandleFunc("/api/v2/snapshot/take/immediate", handlers.SnapshotTakeImmediate).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/snapshot/clone", handlers.SnapshotClone).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/snapshot/all/{res_name}", handlers.SnapshotList).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/snapshot/all/{res_name}/cache", handlers.SnapshotListCache).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/snapshot/destroy", handlers.SnapshotDestroy).Methods(http.MethodDelete)
	r.HandleFunc("/api/v2/snapshot/destroy", handlers.SnapshotDestroy).Methods(http.MethodPost) // additional POST method for the clients that do not support DELETE
	r.HandleFunc("/api/v2/snapshot/rollback", handlers.SnapshotRollback).Methods(http.MethodPost)
	// Metrics
	r.HandleFunc("/api/v2/metrics/vm/{vm_name}", handlers.VmMetrics).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/metrics/jail/{jail_name}", handlers.JailMetrics).Methods(http.MethodGet)
	// Prometheus Metrics and Autodiscovery
	r.HandleFunc("/api/v2/prometheus/autodiscovery/vms", handlers.PrometheusAutoDiscoveryVms).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/prometheus/autodiscovery/vms/use-ips", handlers.PrometheusAutoDiscoveryVmsIps).Methods(http.MethodGet)

	// HA
	r.HandleFunc("/api/v2/carp-ha/ping", handlers.CarpPing).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/carp-ha/backups", handlers.CarpReturnListOfBackups).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/carp-ha/receive-state/{master_hostname}", handlers.CarpReceiveHostState).Methods(http.MethodPost)
	if restConf.HaMode {
		r.HandleFunc("/api/v2/ha/ping", HandlersHA.HandlePing).Methods(http.MethodPost)
		r.HandleFunc("/api/v2/ha/register", HandlersHA.HandleRegistration).Methods(http.MethodPost)
		r.HandleFunc("/api/v2/ha/terminate", HandlersHA.HandleTerminate).Methods(http.MethodPost)
		r.HandleFunc("/api/v2/ha/jail-list", HandlersHA.HandleJailList).Methods(http.MethodGet)
		r.HandleFunc("/api/v2/ha/vm-list", HandlersHA.HandleVmList).Methods(http.MethodGet)
	}

	// Swagger docs
	r.PathPrefix("/api/v2/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/api/v2/swagger.json"), //The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods("GET")
	// Define a route to serve the static file
	r.HandleFunc("/api/v2/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		log.SetStatusCode(http.StatusOK)
		ex, err := os.Executable()
		if err != nil {
			log.Error("could not get the executable path: " + err.Error())
			return
		}
		binPath := filepath.Dir(ex)
		http.ServeFile(w, r, binPath+"/docs/swagger.json")
	})
	// Catch-all route for 404 errors
	r.NotFoundHandler = r.NewRoute().HandlerFunc(handlers.NotFoundHandler).GetHandler()

	bindAddress := fmt.Sprintf("%s:%d", restConf.BindToAddress, restConf.Port)
	logInternal.Info("The REST APIv2 is bound to " + bindAddress)
	http.Handle("/", r)
	srv := &http.Server{
		Addr:         bindAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	err := srv.ListenAndServe()
	if err != nil {
		logInternal.Fatal("could not start the REST API server: " + err.Error())
	}
}
