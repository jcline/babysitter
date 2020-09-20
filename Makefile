.PHONY:all
all: babysitter

.PHONY: babysitter
babysitter:
	go build ./cmd/babysitter

.PHONY: test
test:
	go test ./...

.PHONY: babysitter_bsd
babysitter_bsd:
	GOOS=openbsd GOARCH=amd64 go build ./cmd/babysitter

.PHONY: clean
clean:
	-rm babysitter


