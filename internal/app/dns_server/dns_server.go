package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"HosterCore/internal/pkg/emojlog"
	HosterHost "HosterCore/internal/pkg/hoster/host"

	"github.com/miekg/dns"
)

// Global state vars
var vmInfoList []VmInfoStruct
var jailInfoList []JailInfoStruct
var hostConf HosterHost.HostConfig
var staticRecords []HosterHost.DnsStaticRecord
var upstreamServers []string

var version = "" // version is set by the build system

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

	log.Info("Starting the DNS Server")
	var err error

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP)
	go func() {
		for sig := range signals {
			if sig == syscall.SIGHUP {
				log.Info("Received a reload signal: SIGHUP")
				vmInfoList = getVmsInfo()
				jailInfoList = getJailsInfo()
				loadUpstreamDnsServers()
				loadStaticRecords()
				setResolvConf()

				hostConf, err = HosterHost.GetHostConfig()
				if err != nil {
					log.Fatalf("Failed to read the host config: %s", err.Error())
				}
			}
			if sig == syscall.SIGKILL {
				log.Info("Received a kill signal: SIGKILL, stopping the service now")
				os.Exit(0)
			}
		}
	}()

	loadUpstreamDnsServers()
	loadStaticRecords()
	setResolvConf()

	vmInfoList = getVmsInfo()
	jailInfoList = getJailsInfo()

	hostConf, err = HosterHost.GetHostConfig()
	if err != nil {
		log.Fatalf("Failed to read the host config: %s", err.Error())
		os.Exit(1)
	}

	server := dns.Server{Addr: ":53", Net: "udp"}
	server.Handler = dns.HandlerFunc(handleDNSRequest)

	log.Info("DNS Server is listening on 0.0.0.0:53")
	err = server.ListenAndServe()
	if err != nil {
		emojlog.PrintLogMessage("Failed to start the DNS Server", emojlog.Error)
		os.Exit(1)
	}
}

func loadStaticRecords() {
	// Load host config
	hostConf, err := HosterHost.GetHostConfig()
	if err != nil {
		log.Error("Error loading host config file:" + err.Error())
	}

	// Load static DNS records from the host config
	if len(hostConf.DnsStaticRecords) > 0 {
		staticRecords = append(staticRecords, hostConf.DnsStaticRecords...)
	}
}

// Parses and loads the list of upstream DNS servers from the host config file.
func loadUpstreamDnsServers() {
	// Load host config
	hostConf, err := HosterHost.GetHostConfig()
	if err != nil {
		log.Error("Error loading host config file:" + err.Error())
	}

	// Load upstream DNS servers from the host config
	upstreamServers = []string{}
	reMatchPort := regexp.MustCompile(`.*:\d+`)
	for _, v := range hostConf.DnsServers {
		if reMatchPort.MatchString(v) {
			upstreamServers = append(upstreamServers, v)
		} else {
			upstreamServers = append(upstreamServers, v+":53")
		}
	}

	// If host config doesn't include any servers, use the public ones
	if len(upstreamServers) < 1 {
		upstreamServers = append(upstreamServers, DNS_SRV4_QUAD_NINE)
		upstreamServers = append(upstreamServers, DNS_SRV4_CLOUD_FLARE)
	}

	log.Infof("Loaded these servers from the host config file: %s", upstreamServers)
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)

	var logLine string
	for _, q := range r.Question {
		// Drop any IPv6 record requests
		// TBD: add a config variable that controls this behavior
		clientIP := w.RemoteAddr().String()
		if q.Qtype == dns.TypeAAAA {
			log.Info(fmt.Sprintf("%s -> IPv6 request was ignored: %s", clientIP, q.Name))
			continue
		}

		requestIsVmName := false
		requestIsJailName := false
		requestIsPublic := false
		requestIsStaticRecord := false

		vmListIndex := 0
		jailListIndex := 0
		dnsNameSplit := strings.Split(q.Name, ".")

		if len(dnsNameSplit) > 1 {
			if IsPublicDomain(dnsNameSplit[len(dnsNameSplit)-2]) {
				requestIsPublic = true
			}
		}

		if len(staticRecords) > 0 {
			// Only handle A and CNAME records from the static records for now
			for _, v := range staticRecords {
				if q.Qtype == dns.TypeA && strings.ToUpper(v.Type) == "A" {
					if q.Name == v.Domain+"." || q.Name == v.Domain+"."+hostConf.DnsSearchDomain+"." {
						rr, err := dns.NewRR(q.Name + " IN A " + v.Data)
						if err != nil {
							log.Error("Failed to generate an A record (from the static records): " + err.Error())
							continue
						}
						m.Answer = append(m.Answer, rr)
						logLine = clientIP + " -> " + q.Name + "::." + parseAnswer(m.Answer) + " <- CACHE_HIT::STATIC_A_RECORD"
						log.Info(logLine)
						requestIsStaticRecord = true
					}
				} else if q.Qtype == dns.TypeA && strings.ToUpper(v.Type) == "CNAME" {
					if q.Name == v.Domain+"." || q.Name == v.Domain+"."+hostConf.DnsSearchDomain+"." {
						q.Name = v.Data + "."
						response, server, err := queryExternalDNS(q)
						if err != nil {
							log.Error("Failed to query external DNS:", err)
							continue
						}
						m.Answer = append(m.Answer, response.Answer...)
						logLine = clientIP + " -> " + q.Name + "::." + parseAnswer(m.Answer) + " <- CACHE_MISS::STATIC_CNAME_RECORD::" + server
						log.Info(logLine)
						requestIsStaticRecord = true
					}
				} else if q.Qtype == dns.TypeCNAME && strings.ToUpper(v.Type) == "CNAME" {
					if q.Name == v.Domain+"." || q.Name == v.Domain+"."+hostConf.DnsSearchDomain+"." {
						rr, err := dns.NewRR(q.Name + " IN CNAME " + v.Data)
						if err != nil {
							log.Error("Failed to generate a CNAME record (from the static records): " + err.Error())
							continue
						}
						m.Answer = append(m.Answer, rr)
						logLine = clientIP + " -> " + q.Name + "::." + parseAnswer(m.Answer) + " <- CACHE_HIT::STATIC_CNAME_RECORD"
						log.Info(logLine)
						requestIsStaticRecord = true
					}
				}
			}
		}

		if !requestIsStaticRecord {
			for i, v := range vmInfoList {
				dnsName := dnsNameSplit[0]
				if dnsName == v.vmName {
					if q.Name == v.vmName+"."+hostConf.DnsSearchDomain+"." {
						requestIsPublic = false
					}
					requestIsVmName = true
					vmListIndex = i
				} else if dnsName == strings.ToLower(v.vmName) {
					if q.Name == v.vmName+"."+hostConf.DnsSearchDomain+"." {
						requestIsPublic = false
					}
					requestIsVmName = true
					vmListIndex = i
				}
			}

			for i, v := range jailInfoList {
				dnsName := dnsNameSplit[0]
				if dnsName == v.JailName {
					if q.Name == v.JailName+"."+hostConf.DnsSearchDomain+"." {
						requestIsPublic = false
					}
					requestIsJailName = true
					jailListIndex = i
				} else if dnsName == strings.ToLower(v.JailName) {
					if q.Name == v.JailName+"."+hostConf.DnsSearchDomain+"." {
						requestIsPublic = false
					}
					requestIsJailName = true
					jailListIndex = i
				}
			}

			if requestIsPublic {
				response, server, err := queryExternalDNS(q)
				if err != nil {
					log.Error("Failed to query external DNS:", err)
					continue
				}
				m.Answer = append(m.Answer, response.Answer...)
				logLine = clientIP + " -> " + q.Name + "::." + parseAnswer(m.Answer) + " <- CACHE_MISS::" + server
				log.Info(logLine)
			} else if requestIsVmName {
				rr, err := dns.NewRR(q.Name + " IN A " + vmInfoList[vmListIndex].vmAddress)
				if err != nil {
					log.Error("Failed to create an A record:", err)
					continue
				}
				m.Answer = append(m.Answer, rr)
				logLine = clientIP + " -> " + q.Name + "::." + parseAnswer(m.Answer) + " <- CACHE_HIT::VM"
				log.Info(logLine)
			} else if requestIsJailName {
				rr, err := dns.NewRR(q.Name + " IN A " + jailInfoList[jailListIndex].JailAddress)
				if err != nil {
					log.Error("Failed to create an A record:", err)
					continue
				}
				m.Answer = append(m.Answer, rr)
				logLine = clientIP + " -> " + q.Name + "::." + parseAnswer(m.Answer) + " <- CACHE_HIT::Jail"
				log.Info(logLine)
			} else {
				response, server, err := queryExternalDNS(q)
				if err != nil {
					log.Error("Failed to query external DNS:", err)
					continue
				}
				m.Answer = append(m.Answer, response.Answer...)
				logLine = clientIP + " -> " + q.Name + "::." + parseAnswer(m.Answer) + " <- CACHE_MISS::" + server
				log.Info(logLine)
			}
		}
	}

	err := w.WriteMsg(m)
	if err != nil {
		log.Error("Failed to send the DNS Response:" + err.Error())
	}
}

// Returns a DNS message, a server that returned the response, or an error
func queryExternalDNS(q dns.Question) (*dns.Msg, string, error) {
	c := dns.Client{}
	m := dns.Msg{}
	m.SetQuestion(q.Name, q.Qtype)

	var response *dns.Msg
	var err error
	var responseServer string

	// Try each DNS server until a response is received or all servers fail
	for _, server := range upstreamServers {
		response, _, err = c.Exchange(&m, server)
		if err == nil && response != nil && response.Rcode != dns.RcodeServerFailure {
			// Received a successful response, break the loop
			responseServer = server
			break
		}
	}

	if err != nil {
		return nil, "", err
	}

	return response, responseServer, nil
}

// Regex DNS Answer splitter
var reAnySpaceChar = regexp.MustCompile(`\s+`)

// Parses the DNS answer to only extract the IP address resolved
//
// Used purely for the logging purposes
func parseAnswer(msg []dns.RR) string {
	msgString := fmt.Sprintf("%s", msg)
	splitAnswer := reAnySpaceChar.Split(msgString, -1)
	result := ""
	for i, v := range splitAnswer {
		if i == len(splitAnswer)-1 {
			result = strings.Split(v, "]")[0]
		}
	}
	if result == "[" {
		result = "nil"
	}
	return result
}

func IsPublicDomain(topLevelDomain string) bool {
	for _, v := range publicDomainList {
		if strings.EqualFold(v, topLevelDomain) {
			// the above is the same as this:
			// if strings.ToUpper(v) == strings.ToUpper(topLevelDomain) {
			return true
		}
	}
	return false
}
