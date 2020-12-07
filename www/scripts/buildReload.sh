#!/bin/bash

source /etc/profile
source ~/.profile

go build server.go
service codemirror restart

