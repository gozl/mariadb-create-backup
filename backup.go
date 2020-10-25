package main

import (
	"os"
	"io/ioutil"
	"sort"
	"fmt"
	"path/filepath"
	"time"
	"strings"
	"errors"
	"regexp"
)

func backupDB() error {
	if os.Getuid() != 0 {
		fatalLog("must-runas-root")
	}

	increDir := filepath.Join(backupDir, "incr")
	fullDir := filepath.Join(backupDir, "full")
	runargs := strings.Split(fmt.Sprintf("--user=%s --password=%s --host=%s --port=%d", dbuser, dbpasswd, dbhost, dbport), " ")

	_, errExist := os.Stat(backupDir)
	if os.IsNotExist(errExist) {
		warnLog("non-persist-backup-dir %s", backupDir)
		errMkdir := os.Mkdir(backupDir, 0755)
		if errMkdir != nil {
			return errMkdir
		}
	}

	_, errExist = os.Stat(increDir)
	if os.IsNotExist(errExist) {
		verboseLog("create-incre-dir %s", increDir)
		errMkdir := os.Mkdir(increDir, 0755)
		if errMkdir != nil {
			return errMkdir
		}		
	}

	_, errExist = os.Stat(fullDir)
	if os.IsNotExist(errExist) {
		verboseLog("create-full-dir %s", fullDir)
		errMkdir := os.Mkdir(fullDir, 0755)
		if errMkdir != nil {
			return errMkdir
		}		
	}

	errStatus := testDBStatus(runargs)
	if errStatus != nil {
		return errStatus
	}

	errConnect := testDBConnect(runargs)
	if errConnect != nil {
		return errConnect
	}

	lockPath := filepath.Join(rundir, lockfile)
	verboseLog("create-lockfile %s", lockPath)
	errLock := createLockfile(lockPath)
	if errLock != nil {
		return errLock
	}

	incremental := true

	verboseLog("ls-full-backups")
	fullsubs, errFull := ioutil.ReadDir(fullDir)
	if errFull != nil {
		return errFull
	}
	latestFull, aged := getLatestDir(fullsubs, int64(fullInterval))

	if latestFull == "" {
		verboseLog("first-backup")
		verboseLog("snapshot-type full")
		incremental = false
	} else {
		verboseLog("last-full-backup %s", latestFull)
		if aged {
			verboseLog("last-full-backup-aged")
			verboseLog("snapshot-type full")
			incremental = false
		} else {
			verboseLog("snapshot-type incremental")
			backoff, errIncre := isIncreTooFrequent(increDir, latestFull)
			if errIncre != nil {
				return errIncre
			}
			if backoff {
				fatalLog("snapshot-too-frequent")
			}
		}
	}

	bkdirname := time.Now().Format("2006-01-02_15-04-05")
	var backupPath string
	if !incremental {
		backupPath = filepath.Join(fullDir, bkdirname)
	} else {
		backupPath = filepath.Join(increDir, latestFull, bkdirname)
	}

	verboseLog("snapshot begin")
	verboseLog("target-dir %s", bkdirname)
	var err error
	if incremental {
		err = createSnapshot(backupPath, filepath.Join(fullDir, latestFull), strings.Join(runargs, " "))
	} else {
		err = createSnapshot(backupPath, "", strings.Join(runargs, " "))
	}
	if err != nil {
		return err
	}
	verboseLog("snapshot end")

	if latestFull != "" {
		// make sure there are only <retainCount> entries in fullDir n increDir
		err = enforceRetainCount(fullDir, increDir)
		if err != nil {
			return err
		}
	}
	
	return nil
}

func backupExitHook() error {
	lockPath := filepath.Join(rundir, lockfile)
	verboseLog("remove-lockfile %s", lockPath)
	err := rmfile(lockPath)
	if err != nil {
		return err
	}

	return nil
}

func isIncreTooFrequent(incredir, parentName string) (bool, error) {
	verboseLog("ls-incre-backups")
	fis, err := ioutil.ReadDir(incredir)
	if err != nil {
		return false, err
	}

	if len(fis) == 0 {
		verboseLog("incre-dir-empty")
		return false, nil
	}

	parentDir := ""
	for _, r := range fis {
		if !r.IsDir() {
			continue
		}
		if r.Name() == parentName {
			parentDir = filepath.Join(incredir, parentName)
			break
		}
	}

	if parentDir == "" {
		verboseLog("first-incre-backup")
		return false, nil
	}

	fis, err = ioutil.ReadDir(parentDir)
	_, aged := getLatestDir(fis, int64(increInterval))
	return !aged, nil
}

func createSnapshot(targetDir, parentDir string, runargs string) error {
	// create dir
	verboseLog("create-dir %s", targetDir)
	errMkdir := os.MkdirAll(targetDir, 0750)
	if errMkdir != nil {
		return errMkdir
	}

	// 100=mysql, 0=root
	verboseLog("set-dir-owner %s owner=100:0", targetDir)
	errPerm := os.Chown(targetDir, 100, 0)
	if errPerm != nil {
		return errPerm
	}
	if parentDir != "" {
		// /data/backup/incr/2020-09-08_06-16-04/2020-09-08_06-26-04/
		//                            +---> this needs to be owned by mysql too
		errPerm = os.Chown(filepath.Dir(targetDir), 100, 0)
		if errPerm != nil {
			return errPerm
		}
	}

	incremental := false
	if parentDir != "" {
		incremental = true
	}

	// run mariabackup
	bkupCmd := "mariabackup --backup %s --extra-lsndir=\"%s\" --stream=xbstream | gzip > \"%s/backup.stream.gz\""
	if incremental {
		parentDirParam := fmt.Sprintf("--incremental-basedir=\"%s\"", parentDir)
		bkupCmd = fmt.Sprintf(bkupCmd, runargs + " " + parentDirParam, targetDir, targetDir)	
	} else {
		bkupCmd = fmt.Sprintf(bkupCmd, runargs, targetDir, targetDir)
	}
	
	passwdArgRE := regexp.MustCompile(`\s--password=\S+\s`)
	bkupCmdReduct := passwdArgRE.ReplaceAllString(bkupCmd, " --password=***** ")
	verboseLog("exec :: s6-setuidgid mysql -c '%s'", bkupCmdReduct)
	binO, binE, err := execCmd("s6-setuidgid", "", []string{ "mysql", "sh", "-c", bkupCmd }, int(backupTimeout))

	logErr := ioutil.WriteFile(filepath.Join(targetDir, "stdout.log"), []byte(binO), 0640)
	if logErr != nil {
		errorLog("write-file %s :: %v", "stdout.log", logErr)
	}
	logErr = ioutil.WriteFile(filepath.Join(targetDir, "stderr.log"), []byte(binE), 0640)
	if logErr != nil {
		errorLog("write-file %s :: %v", "stderr.log", logErr)
	}

	if err != nil {
		errorLog("exec-cmd :: %s", bkupCmd)
		dumpCmdResult(binO, binE)
		return err
	}

	verboseLog("mariabackup-complete-show-log")
	if binO != "" {
		dumpCmdResult(binO, binE)
	} else {
		// mariabackup seems to write to stderr
		dumpCmdResult(binE, "")
	}

	// fix perms
	verboseLog("read-dir %s", targetDir)
	fis, err := ioutil.ReadDir(targetDir)
	if err != nil {
		return err
	}

	hasPermErr := false
	for _, fi := range fis {
		fPath := filepath.Join(targetDir, fi.Name())
		// make dir readonly to mysql:root
		verboseLog("protect-file %s", fPath)

		err = os.Chmod(fPath, 0440)
		if err != nil {
			errorLog("chmod :: %v", err)
			hasPermErr = true
		}

		err = os.Chown(fPath, 100, 0)
		if err != nil {
			errorLog("chown :: %v", err)
			hasPermErr = true
		}
	}

	if hasPermErr {
		return errors.New("fsperm-failed")
	}
	return nil
}

func enforceRetainCount(fulldir, incredir string) error {
	names := []string{}
	fis, err := ioutil.ReadDir(fulldir)
	if err != nil {
		return err
	}

	for _, r := range fis {
		if !r.IsDir() {
			continue
		}
		names = append(names, r.Name())
	}
	sort.Strings(names)

	if len(names) <= int(retainCount) {
		return nil
	}

	delCount := len(names) - int(retainCount)
	delNames := names[0:delCount]
	verboseLog("delete-backups-count %d", delCount)

	hasDelErr := false
	for _, n := range delNames {
		verboseLog("delete-full-backup %s", n)
		err = os.RemoveAll(filepath.Join(fulldir, n))
		if err != nil {
			errorLog("del-full-backup :: %v", err)
			hasDelErr = true
		}

		verboseLog("delete-incre-backups %s", n)
		os.RemoveAll(filepath.Join(incredir, n))
		if err != nil {
			errorLog("del-incre-backup :: %v", err)
			hasDelErr = true
		}
	}

	if hasDelErr {
		return errors.New("fsdel-failed")
	}
	return nil
}

func getLatestDir(fi []os.FileInfo, minInterval int64) (string, bool) {
	if fi == nil || len(fi) == 0 {
		return "", false
	}

	aged := false
	names := []string{}
	for _, r := range fi {
		if !r.IsDir() {
			continue
		}

		names = append(names, r.Name())
	}
	sort.Strings(names)
	latest := names[len(names) - 1]

	for _, r := range fi {
		if r.Name() == latest {
			currentTime := time.Now()
			verboseLog("last-backup-time :: %v", r.ModTime())
			verboseLog("current-time :: %v", currentTime)

			timeDiff := currentTime.Unix() - r.ModTime().Unix()
			verboseLog("elapsed-seconds %d", timeDiff)

			if timeDiff > minInterval {
				aged = true
				verboseLog("backup-expired-seconds %d", timeDiff - minInterval)
			} else {
				verboseLog("time-to-backup-expiry-seconds %d", minInterval - timeDiff)
			}
			break
		}
	}
	return latest, aged
}