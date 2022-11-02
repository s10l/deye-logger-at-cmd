default: build

all: build

build: clean
	@cd src && go build -o ../build/main


clean:
	@rm -rf ./build
