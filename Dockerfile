FROM dockerguys/alpine:3.12

RUN set -ex ; \
	apk-install git golang ; \
	mkdir -p /go/bin ; \
	mkdir -p /go/src/github.com/gozl ; \
	cd /go/src/github.com/gozl ; \
	git clone https://github.com/gozl/mariadb-create-backup.git ; \
	cd mariadb-create-backup ; \
	go build -v -o /go/bin/mariadb-create-backup
