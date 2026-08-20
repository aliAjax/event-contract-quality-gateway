.PHONY: fmt vet test build run smoke stop count
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')
vet:
	go vet ./...
test:
	go test ./...
build:
	go build ./cmd/gateway ./cmd/worker
run:
	go run ./cmd/gateway
smoke:
	./scripts/smoke.sh
stop:
	pkill -f 'event-contract-quality-gateway|cmd/gateway' || true
count:
	./scripts/count-go-lines.sh
