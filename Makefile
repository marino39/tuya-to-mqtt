SHELL := /bin/bash

test:
	@go install github.com/mfridman/tparse@latest
	@set -o pipefail && go test ./... -json -cover | tparse -all

test-ci:
	@go test ./...