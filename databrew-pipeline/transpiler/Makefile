.PHONY: build install test clean api

build:
	go build -o bin/dbtp ./cmd/dbtp

api:
	go build -o bin/apiserver ./cmd/apiserver

install:
	go install ./cmd/dbtp

test: build
	./bin/dbtp ../examples/single-step.json --yaml | head -5
	@echo "---"
	./bin/dbtp ../examples/three-step-data-pass.json | python3 -c "import sys,json; d=json.load(sys.stdin); print('Templates:', len(d['spec']['templates'])); print('DAG tasks:', len(d['spec']['templates'][-1]['dag']['tasks']))"

clean:
	rm -rf bin/
