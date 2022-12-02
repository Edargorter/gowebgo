#!/bin/bash

sudo fuser -k 8081/tcp
#trap "No requests yet."
rm requests/* 2> /dev/null
#strace go run server.go 
go run gowebgo.go 
