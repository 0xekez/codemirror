#!/bin/bash

source /etc/profile
source ~/.profile

go build server.go

/bin/systemctl restart codemirror

