mariadb-create-backup
=====================
A simple utility to take periodic backups for mariadb.

This app was originally ported from https://github.com/omegazeng/run-mariabackup :-)

Requirements
------------
This utility runs on Linux only. It assumes that the following are installed:
- s6-setuidgid
- mariabackup
- mysql
- db user with backup privileges (see https://mariadb.com/kb/en/mariabackup-overview/#authentication-and-privileges)

Backup directory layout
-----------------------
Base directory is `/data/backup` by default.

Full backups are saved to `<base>/full/<time>/`. When run, the app decides whether to take another 
full backup based on the modtime of the latest full backup dir. You can customize the interval using 
option `-i`.

If a full backup is not required, it will consider whether an incremental backup should be done. Incremental 
backups are saved to `<base>/incr/<time>/<time2>/`, where `<time>` is the folder name that contains a full 
backup (aka `<base>/full/<time>/`). You can restore the database to the time an incremental backup is taken 
by having **BOTH** the `<base>/full/<time>/` and `<base>/incr/<time>/<time2>/` folders. The app will refuse 
to take an incremental backup if a previous one was done recently. You can customize the interval using 
option `-I`.

The app will also delete old backups. Use `-c` to control how many full backups should exist. When a full 
backup is deleted, its associating incremental backups are deleted too.

Build Instructions
------------------
Just build in docker:

```bash
docker build -v ~/bin:/go/bin ./
ls ~/bin/
```








