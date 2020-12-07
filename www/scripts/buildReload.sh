#!/bin/bash

go install
go build server.go
service codemirror restart

