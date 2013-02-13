
all: build

build:
	go build -o bin/nerdtalk src/nerdtalk/*.go

clean:
	rm -r bin

run: build
	bin/nerdtalk