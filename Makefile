
run: fmt vet
	go run ./main/main.go

fmt:
	go fmt ./pkg/... ./main/...

vet:
	go vet ./pkg/... ./main/...

