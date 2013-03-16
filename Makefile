all: build

build:
ifeq ($(OS),Windows_NT)
	go build -o bin/nerdtalk.exe src/nerdtalk/*.go
else
	go build -o bin/nerdtalk.exe src/nerdtalk/*.go
endif

clean:
	rm -r bin

run: build
	bin/nerdtalk