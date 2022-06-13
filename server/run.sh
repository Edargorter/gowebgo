#!/bin/bash

sudo fuser -k 8081/tcp
#trap "No requests yet."
rm requests/*
#strace go run server.go 
go run server.go 
