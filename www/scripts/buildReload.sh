#!/bin/bash

go install
go build server.go
sudo service codemirror restart
