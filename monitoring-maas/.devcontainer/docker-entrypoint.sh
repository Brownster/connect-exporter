#!/bin/sh
set -e
go get golang.org/x/tools/gopls@latest
go mod download
mkdir -p /root/.ssh
mkdir -p /root/.gnupg

cp -Rf /tmp/.ssh/* /root/.ssh/.
cp -Rf /tmp/.gnupg/* /root/.gnupg/.

chmod 700 /root/.ssh
chmod 644 /root/.ssh/id_rsa.pub
chmod 600 /root/.ssh/id_rsa
