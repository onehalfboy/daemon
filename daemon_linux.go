// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

// Package daemon linux version
package daemon

import (
	"os"
)

// Get the daemon properly
func newDaemon(name, port string, version string, description string, dependencies []string) (Daemon, error) {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return &systemDRecord{name, port, version, description, dependencies}, nil
	}
	return &systemVRecord{name, port, version, description, dependencies}, nil
}

// Get executable path
func execPath() (string, error) {
	return os.Readlink("/proc/self/exe")
}
