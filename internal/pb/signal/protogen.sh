#!/usr/bin/env bash

protoc -I=. --go_out=. --go_opt=paths=source_relative signal.proto
