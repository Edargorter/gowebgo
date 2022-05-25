#!/bin/bash

sudo fuser -k 8081/tcp
#trap "No requests yet."
rm requests/*
go run server.go 
