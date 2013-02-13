
all: build

build:
	go build -o bin/nerdtalk .../src/nerdtalk

clean:
	rm -r bin

run: build
	bin/nerdtalk