#!/bin/bash

source /etc/profile
source ~/.profile

GO111MODULE=auto go get
go build server.go
service codemirror restart

