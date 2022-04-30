#!/bin/bash

sudo fuser -k 8081/tcp
rm requests/*
go run server.go
