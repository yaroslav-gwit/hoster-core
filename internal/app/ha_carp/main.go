package main

import (
	CarpUtils "HosterCore/internal/app/ha_carp/utils"
	"fmt"
	"net"
	"os"
)

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

	// Remove the old socket if it exists
	if _, err := os.Stat(CarpUtils.SOCKET_FILE); err == nil {
		os.Remove(CarpUtils.SOCKET_FILE)
	}

	// Create the Unix socket listener
	listener, err := net.Listen("unix", CarpUtils.SOCKET_FILE)
	if err != nil {
		log.Fatalf("Error creating Unix socket: %v", err)
	}
	defer listener.Close()

	log.Infof("Server listening on %s\n", CarpUtils.SOCKET_FILE)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("Error accepting connection:", err)
			continue
		}

		// Handle the connection in a new goroutine
		go handleConnection(conn)
	}

	// out, err := CarpUtils.ParseIfconfig()
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }

	// fmt.Println(out)
}
