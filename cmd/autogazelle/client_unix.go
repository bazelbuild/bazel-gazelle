// +build darwin dragonfly freebsd js,wasm linux nacl netbsd openbsd solaris

/* Copyright 2018 The Bazel Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

// runClient performs the main work of the client. It attempts to connect
// to the server via a UNIX-domain socket. If the server is not running,
// it starts the server and tries again. The server does all the work, so
// the client just waits for the server to complete, then exits.
func runClient() error {
	startTime := time.Now()
	conn, err := net.Dial("unix", *socketPath)
	if err != nil {
		if err := startServer(); err != nil {
			return fmt.Errorf("error starting server: %v", err)
		}
		for retry := 0; retry < 3; retry++ {
			conn, err = net.Dial("unix", *socketPath)
			if err == nil {
				break
			}
			// Wait for server to start listening.
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			return fmt.Errorf("failed to connect to server: %v", err)
		}
	}
	defer conn.Close()

	if _, err := io.Copy(os.Stderr, conn); err != nil {
		log.Print(err)
	}

	elapsedTime := time.Since(startTime)
	log.Printf("ran gazelle in %.3f s", elapsedTime.Seconds())
	return nil
}
