.PHONY: build wasm serve-wasm clean

# Ebitengine v2.10+ on macOS: full Metal/UI path without CGO (purego/objc).
export CGO_ENABLED := 0

DIST := dist/wasm
GOROOT := $(shell go env GOROOT)
# Go 1.21+: lib/wasm; older toolchains used misc/wasm
WASM_EXEC := $(shell test -f "$(GOROOT)/lib/wasm/wasm_exec.js" && echo "$(GOROOT)/lib/wasm/wasm_exec.js" || echo "$(GOROOT)/misc/wasm/wasm_exec.js")

build:
	mkdir -p dist
	go build -o dist/logos .

wasm:
	mkdir -p $(DIST)
	GOOS=js GOARCH=wasm go build -trimpath -o $(DIST)/game.wasm .
	cp "$(WASM_EXEC)" $(DIST)/wasm_exec.js
	cp wasm/index.html $(DIST)/index.html
	cd $(DIST) && zip -q -FS logos.zip index.html wasm_exec.js game.wasm

serve-wasm: wasm
	cd $(DIST) && python3 -m http.server 8080

clean:
	rm -rf dist/
