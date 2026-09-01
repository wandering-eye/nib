.PHONY: release clean install
dist/nib: clean
	go build -o dist/nib cmd/nib/main.go
	go build -o dist/dstask-import cmd/dstask-import/main.go

release:
	./do-release.sh

clean:
	rm -rf dist

install:
	cp dist/nib /usr/local/bin
	cp dist/dstask-import /usr/local/bin

test:
	go test -v ./...
	./integrationtest.sh | cat  # cat -- no tty, no confirmations

lint:
	golangci-lint run

update_deps:
	go get
