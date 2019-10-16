#!/usr/bin/env sh

protoc service.proto --go_out=plugins=grpc:.
