// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

package daemon

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

// systemVRecord - standard record (struct) for linux systemV version of daemon package
type systemVRecord struct {
	name         string
	port         string
	version      string
	description  string
	dependencies []string
}

// Standard service path for systemV daemons
func (linux *systemVRecord) servicePath() string {
	return "/etc/init.d/" + linux.name
}

// Is a service installed
func (linux *systemVRecord) isInstalled() bool {

	if _, err := os.Stat(linux.servicePath()); err == nil {
		return true
	}

	return false
}

// Check service is running
func (linux *systemVRecord) checkRunning() (string, bool) {
	output, err := exec.Command("service", linux.name, "status").Output()
	if err == nil {
		if matched, err := regexp.MatchString("running", string(output)); err == nil && matched {
			if matched, err := regexp.MatchString(linux.name, string(output)); err == nil && matched {
				reg := regexp.MustCompile("pid  ([0-9]+)")
				data := reg.FindStringSubmatch(string(output))
				if len(data) > 1 {
					return "Service " + linux.name + " (pid  " + data[1] + ") is running...", true
				}
				return "Service " + linux.name + " is running...", true
			}
		}
	}

	return "Service " + linux.name + " is stopped", false
}

// Install the service
func (linux *systemVRecord) Install(args ...string) (string, error) {
	installAction := "Install " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return installAction + failed, err
	}

	srvPath := linux.servicePath()

	if linux.isInstalled() {
		return installAction + failed, ErrAlreadyInstalled
	}

	file, err := os.Create(srvPath)
	if err != nil {
		return installAction + failed, err
	}
	defer file.Close()

	execPatch, err := executablePath(linux.name)
	if err != nil {
		return installAction + failed, err
	}

	templ, err := template.New("systemVConfig").Parse(systemVConfig)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Port, Version, Description, Path, Args string
		}{linux.name, linux.port, linux.version, linux.description, execPatch, strings.Join(args, " ")},
	); err != nil {
		return installAction + failed, err
	}

	if err := os.Chmod(srvPath, 0755); err != nil {
		return installAction + failed, err
	}

	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Symlink(srvPath, "/etc/rc"+i+".d/S87"+linux.name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Symlink(srvPath, "/etc/rc"+i+".d/K17"+linux.name); err != nil {
			continue
		}
	}

	return installAction + success, nil
}

// Remove the service
func (linux *systemVRecord) Remove() (string, error) {
	removeAction := "Removing " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return removeAction + failed, err
	}

	if !linux.isInstalled() {
		return removeAction + failed, ErrNotInstalled
	}

	if err := os.Remove(linux.servicePath()); err != nil {
		return removeAction + failed, err
	}

	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Remove("/etc/rc" + i + ".d/S87" + linux.name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Remove("/etc/rc" + i + ".d/K17" + linux.name); err != nil {
			continue
		}
	}

	return removeAction + success, nil
}

// Start the service
func (linux *systemVRecord) Start() (string, error) {
	startAction := "Starting " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return startAction + failed, err
	}

	if !linux.isInstalled() {
		return startAction + failed, ErrNotInstalled
	}

	if _, ok := linux.checkRunning(); ok {
		return startAction + failed, ErrAlreadyRunning
	}

	if err := exec.Command("service", linux.name, "start").Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (linux *systemVRecord) Stop() (string, error) {
	stopAction := "Stopping " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return stopAction + failed, err
	}

	if !linux.isInstalled() {
		return stopAction + failed, ErrNotInstalled
	}

	if _, ok := linux.checkRunning(); !ok {
		return stopAction + failed, ErrAlreadyStopped
	}

	if err := exec.Command("service", linux.name, "stop").Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (linux *systemVRecord) Status() (string, error) {

	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if !linux.isInstalled() {
		return "Status could not defined", ErrNotInstalled
	}

	statusAction, _ := linux.checkRunning()

	return statusAction, nil
}

// Path - Get service path
func (linux *systemVRecord) ExecPath(serviceName string) (string, error) {

	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if !linux.isInstalled() {
		return "", ErrNotInstalled
	}

	if serviceName == "" {
		serviceName = linux.name
	}
	output, err := exec.Command("service", serviceName, "execpath").Output()

	return string(output), err
}

// Restart the service
func (linux *systemVRecord) Restart() (string, error) {
	startAction := "Restarting " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return startAction + failed, err
	}

	if !linux.isInstalled() {
		return startAction + failed, ErrNotInstalled
	}

	if err := exec.Command("service", linux.name, "restart").Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

var systemVConfig = `#! /bin/sh
#
#       /etc/rc.d/init.d/{{.Name}}
#
#       Starts {{.Name}} as a daemon
#
# chkconfig: 2345 87 17
# description: Starts and stops a single {{.Name}} instance on this system

### BEGIN INIT INFO
# Provides: {{.Name}} 
# Required-Start: $network $named
# Required-Stop: $network $named
# Default-Start: 2 3 4 5
# Default-Stop: 0 1 6
# Short-Description: This service manages the {{.Description}}.
# Description: {{.Description}}
### END INIT INFO

#
# Source function library.
#
if [ -f /etc/rc.d/init.d/functions ]; then
    . /etc/rc.d/init.d/functions
fi

exec="{{.Path}}"
servname="{{.Description}}"
port="{{.Port}}"
version="{{.Version}}"

proc="{{.Name}}"
pidfile="/var/run/$proc.pid"
lockfile="/var/lock/subsys/$proc"
stdoutlog="/var/log/$proc.log"
stderrlog="/var/log/$proc.err"

privilegeCheck() {
    testFile="/var/tmp/${proc}_testPriv.t"
    touch $testFile
    chown root:root $testFile 2>/dev/null
    if [ $? != 0 ]; then
        rm -f $testFile
        echo "Error:  You must have root user privileges. Possibly using 'sudo' command should help"
        exit 1
    fi
    rm -f $testFile
    if [ $? != 0 ]; then
        echo "Error:  You must have root user privileges. Possibly using 'sudo' command should help"
        exit 1
    fi
}

# root or sudo command
privilegeCheck

[ -d $(dirname $lockfile) ] || mkdir -p $(dirname $lockfile)

[ -e /etc/sysconfig/$proc ] && . /etc/sysconfig/$proc

start() {
    [ -x $exec ] || exit 5

    if [ -f $pidfile ]; then
        if ! [ -d "/proc/$(cat $pidfile)" ]; then
            rm $pidfile
            if [ -f $lockfile ]; then
                rm $lockfile
            fi
        fi
    fi

    if ! [ -f $pidfile ]; then
        printf "Starting $servname:\t"
        if [ $? != 0 ]; then
            echo -n "Starting $servname:     "
        fi
        echo "$(date)" >> $stdoutlog
        $exec {{.Args}} >> $stdoutlog 2>> $stderrlog &
        echo $! > $pidfile
        touch $lockfile
        success 2>/dev/null
        if [ $? != 0 ]; then
            echo "[ OK ]"
        else
            echo
        fi
    else
        # failure
        echo
        printf "$pidfile still exists...\n"
        if [ $? != 0 ]; then
            echo "$pidfile still exists..."
        fi
        exit 7
    fi
}

stop() {
    printf "Stopping $servname:\t"
    if [ $? != 0 ]; then
        echo -n "Stopping $servname:     "
    fi
    kill -9 $(cat $pidfile) && rm -f $pidfile
    retval=$?
    [ $retval -eq 0 ] && rm -f $lockfile
    if [ $? != 0 ]; then
        failure 2>/dev/null
        if [ $? != 0 ]; then
            echo "[ FAILED ]"
        else
            echo
        fi
    else
        success 2>/dev/null
        if [ $? != 0 ]; then
            echo "[ OK ]"
        else
            echo
        fi
    fi
    return $retval
}

clear() {
    rm -f $pidfile
    rm -f $lockfile
}

restart() {
    stop >/dev/null 2>/dev/null
    start
}

rh_status() {
        pidCmd=$(lsof -i${port} | awk '{print $2;}' | sed -n 2p)
        if [ ! -f "$pidfile" ] && [ -z "$pidCmd" ]; then
            echo "$servname is stopped";
            return 1;
        fi
        pid=$(cat $pidfile);
        if [ "$pidCmd" != "$pid" ]; then
            if [ ! -z "$pid" ]; then
                kill -9 $pid
            fi
            if [ ! -z "$pidCmd" ]; then
                echo "$pidCmd" > "$pidfile";
                touch $lockfile
                echo "$servname is running [pid  $(cat $pidfile)]";
                return 0;
            fi
            clear
            echo "$servname is stopped";
            return 1;
        fi
        echo "$servname is running [pid  $pid]";
        return 0;
}

rh_status_q() {
    rh_status >/dev/null 2>&1
}

case "$1" in
    start)
        rh_status_q && exit 0
        $1
        ;;
    stop)
        rh_status_q || exit 0
        $1
        ;;
    restart)
        $1
        ;;
    status)
        rh_status
        ;;
    version)
        echo $version
        ;;
    description)
        echo $servname
        ;;
    execpath)
        echo $exec
        ;;
    *)
        echo "Usage: $0 {start|stop|status|restart|version|description|execpath}"
        exit 2
esac

exit $?
`
