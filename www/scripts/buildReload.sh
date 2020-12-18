#!/bin/bash

source /etc/profile
source ~/.profile

go build server.go

sudo /bin/systemctl restart codemirror
