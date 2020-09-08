package main

import (
	"flag"
	"os"
	"fmt"
	"path/filepath"
)

const (
	appName = "mariadb-backup"
	appVer = "1.0.0"
	appDesc = "MariaDB backup helper"
)

var (
	dbuser string
	dbpasswd string
	dbhost string
	dbport uint
	backupDir string
	fullInterval uint
	increInterval uint
	retainCount uint
	backupTimeout uint
	verbose bool
	rundir string

	lockfile = "mariadb-backup.lock"
)

func init() {
	flag.StringVar(&dbuser, "u", "backup", "Database username")
	flag.StringVar(&dbpasswd, "p", "", "Database password")
	flag.StringVar(&dbhost, "H", "localhost", "Database server hostname")
	flag.UintVar(&dbport, "P", 3306, "Database server port")
	flag.StringVar(&backupDir, "d", "/data/backup", "Backup base directory")
	flag.StringVar(&rundir, "r", "/tmp", "Run directory")
	flag.UintVar(&fullInterval, "i", 86400, "Interval in seconds between each full backup")
	flag.UintVar(&increInterval, "I", 3600, "Interval in seconds between each incremental backup")
	flag.UintVar(&retainCount, "c", 10, "Number of backups to retain")
	flag.UintVar(&backupTimeout, "t", 600, "Number of seconds before backup timeout")
	flag.BoolVar(&verbose, "v", false, "Verbose mode")

	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "%s %s () %s\n", appName, appVer, appDesc)
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintf(os.Stdout, "Usage: %s -u <user> -p <password> -h <host> -p <port> -d <dir>\n", os.Args[0])
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "Parameters:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintf(os.Stdout, "%s will take a full backup at <d>/full/<t1> or <d>/incre/<t1>/<t2> depending \non <i>. It will delete all of <d>/full/<tX> and <d>/incre/<tX> such that there are \nexactly <c> number of directories under <d>/full and <d>/incre.\n", appName)
	}
}

func main() {
	lockPath := filepath.Join(rundir, lockfile)
	if fileExists(lockfile) {
		fmt.Printf("FATAL! lockfile-exists %s", lockPath)
		os.Exit(1)
	}

	// parse flags or exit
	flag.Parse()

	// sanity check
	if dbport < 1 || dbport > 65535 {
		fatalLog("param-out-of-range -P %d", dbport)
	}
	if fullInterval < 60 {
		fatalLog("param-out-of-range -i %d", fullInterval)
	}
	if retainCount < 1 {
		fatalLog("param-out-of-range -c %d", retainCount)
	}
	if backupTimeout < 30 {
		fatalLog("param-out-of-range -t %d", backupTimeout)
	}

	defer func() {
		verboseLog("cleanup start")
		if err := backupExitHook(); err != nil {
			fmt.Printf("FATAL! cleanup-error :: %v\n", err)
			os.Exit(1)
		}
		verboseLog("cleanup end")
		verboseLog("goodbye")
	}()

	verboseLog("backup start")
	if err := backupDB(); err != nil {
		fatalLog("backup-error :: %v", err)
	}
	verboseLog("backup end")
}
