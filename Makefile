
default: all

build:
	@bash --norc -i ./scripts/compileReceptor.sh

plugins:
	@bash --norc -i ./scripts/compilePlugins.sh

test:
	@bash --norc -i ./scripts/runTests.sh

all: test build plugins

runexample:
	./bin/receptor -config="./receptor_example.conf.json" -plugins="./bin/plugins/"

clean:
	@rm -rf -- "./bin"
	@rm -- "./build.log"

.PHONY: build test plugins all clean