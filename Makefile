test:
	@go install github.com/mfridman/tparse@latest
	@set -o pipefail && go test ./... -json | tparse -all

test-ci:
	@go test ./...