.PHONY:all
all: babysitter sit

.PHONY: babysitter
babysitter:
	go build ./cmd/babysitter

.PHONY: sit
sit:
	go build ./cmd/sit

.PHONY: test
test:
	go test ./...

.PHONY: bench
bench:
	go test -bench=. ./...


.PHONY: babysitter_bsd
babysitter_bsd:
	GOOS=openbsd GOARCH=amd64 go build ./cmd/babysitter

.PHONY: clean
clean:
	-rm babysitter


