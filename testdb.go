package main

import (
	"errors"
	"strings"
)

func testDBStatus(runargs []string) error {
	statusO, statusE, errStatus := execCmd("mysqladmin", "", append(runargs, "status"), 10)

	if errStatus != nil {
		errorLog("exec-cmd mysqladmin")
		dumpCmdResult(statusO, statusE)
		return errStatus
	}

	if statusE != "" {
		errorLog("exec-cmd-stderr mysqladmin")
		dumpCmdResult(statusO, statusE)
		return errors.New("execute-error mysqladmin")
	}

	dumpCmdResult(statusO, "")

	if !strings.Contains(statusO, "Uptime:") {
		errorLog("unexpected-db-status mysqladmin")
		return errors.New("db-status-error")
	}
	
	verboseLog("db-status success")
	return nil
}

func testDBConnect(runargs []string) error {
	connectO, connectE, errConnect := execCmd("mysql", "exit", append(runargs, "-s"), 10)

	if errConnect != nil {
		errorLog("exec-cmd mysql")
		dumpCmdResult(connectO, connectE)
		return errConnect
	}

	if connectE != "" {
		errorLog("exec-cmd-stderr mysql")
		dumpCmdResult(connectO, connectE)
		return errors.New("execute-error mysql")
	}

	if connectO != "" {
		dumpCmdResult(connectO, "")
		return errors.New("db-connect-error")
	}

	verboseLog("db-connect success")
	return nil
}
