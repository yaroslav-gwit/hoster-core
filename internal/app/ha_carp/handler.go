package main

import (
	CarpUtils "HosterCore/internal/app/ha_carp/utils"
	"encoding/json"
	"net"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	var buf [512]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		log.Error("Error reading from connection:", err)
		return
	}

	// First, unmarshal the base struct to detect the payload type
	var base CarpUtils.BasePayload
	err = json.Unmarshal(buf[:n], &base)
	if err != nil {
		log.Error("Error unmarshalling JSON:", err)
		return
	}

	// Switch on the type field to unmarshal into the correct struct
	switch base.Type {
	case "host_add":
		var payload CarpUtils.HostInfo
		err = json.Unmarshal(buf[:n], &payload)
		if err != nil {
			log.Error("Error unmarshalling Hosts Payload:", err)
			return
		}

		mutexHosts.Lock()
		hosts = append(hosts, payload)
		mutexHosts.Unlock()
		log.Infof("Received Hosts Payload: %+v\n", payload)

	case "backup_add":
		var payload CarpUtils.BackupInfo
		err = json.Unmarshal(buf[:n], &payload)
		if err != nil {
			log.Error("Error unmarshalling Backups Payload:", err)
			return
		}

		mutexBackups.Lock()
		backups = append(backups, payload)
		mutexBackups.Unlock()
		log.Infof("Received Backups Payload: %+v\n", payload)

	default:
		log.Warn("Unknown payload type:", base.Type)
	}

	// Send a response back
	response := CarpUtils.SocketResponse{
		Success: true,
	}
	respBytes, _ := json.Marshal(response)
	conn.Write(respBytes)
}
