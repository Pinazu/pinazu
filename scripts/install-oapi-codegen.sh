#!/bin/bash

# Install oapi-codegen
go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# Verify installation
go tool oapi-codegen -version