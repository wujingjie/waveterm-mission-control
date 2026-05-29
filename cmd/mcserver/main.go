// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

func generateAuthKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func getDataHome() string {
	if v := os.Getenv("MC_DATA_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/mc"
	}
	return filepath.Join(home, ".mc")
}

func getHost() string {
	if v := os.Getenv("MC_HOST"); v != "" {
		return v
	}
	return "127.0.0.1"
}

func getPort() string {
	if v := os.Getenv("MC_PORT"); v != "" {
		return v
	}
	return "3001"
}

func main() {
	authKey := os.Getenv("MC_AUTH_KEY")
	if authKey == "" {
		var err error
		authKey, err = generateAuthKey()
		if err != nil {
			log.Fatalf("failed to generate auth key: %v", err)
		}
		log.Printf("MC_AUTH_KEY not set, generated ephemeral key\n")
	}

	dataHome := getDataHome()
	host := getHost()
	port := getPort()
	addr := host + ":" + port

	if err := initDB(dataHome); err != nil {
		log.Fatalf("initDB: %v", err)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}

	startStalenessWatcher()

	router := makeRouter(authKey)

	// Signal readiness to Electron — this line is parsed by emain-mcsrv.ts
	fmt.Fprintf(os.Stderr, "MCSRV-ESTART api:%s\n", addr)

	log.Printf("mcsrv listening on %s\n", addr)
	if err := http.Serve(listener, router); err != nil {
		log.Fatalf("http serve: %v", err)
	}
}
