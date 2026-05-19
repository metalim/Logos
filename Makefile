.PHONY: build wasm serve-wasm build-win build-mac build-mac-arm64 build-mac-amd64 dist clean FORCE

# Ebitengine v2.10+ on macOS: full Metal/UI path without CGO (purego/objc). Cross-
# compiling to Windows and darwin/amd64 also uses the CGO-less backend, so every
# target in this Makefile works straight from a plain Mac host — no mingw, no SDK.
export CGO_ENABLED := 0

BIN      := logos
PKG      := ./cmd/logos
DIST     := dist
WASM_DIR := $(DIST)/wasm
# Ludum Dare 59 freeze; bump only if you re-tag the jam build.
JAM_COMMIT := 44f60e8
JAM_STAMP  := $(WASM_DIR)/.jam-built-$(JAM_COMMIT)
WIN_DIR  := $(DIST)/windows
MAC_DIR  := $(DIST)/mac

# Fingerprint of all local inputs (Go sources, embedded assets, module lock).
# Filenames with spaces are hashed inside the shell recipe — not listed as make deps.
INPUTS_SUM := $(DIST)/.inputs.sha256

GOROOT    := $(shell go env GOROOT)
# Go 1.21+: lib/wasm; older toolchains used misc/wasm
WASM_EXEC := $(shell test -f "$(GOROOT)/lib/wasm/wasm_exec.js" && echo "$(GOROOT)/lib/wasm/wasm_exec.js" || echo "$(GOROOT)/misc/wasm/wasm_exec.js")

# Distribution-only linker flags: strip symbol and DWARF tables so shipped zips
# are smaller. The plain `build` target (dev loop) leaves symbols in for usable
# panics; everything we actually upload to itch.io is stripped.
DIST_LDFLAGS := -s -w
# Windows GUI apps must link with -H=windowsgui so a spare console window
# doesn't pop up next to the game window when the player double-clicks the exe.
WIN_LDFLAGS  := $(DIST_LDFLAGS) -H=windowsgui

$(INPUTS_SUM): FORCE
	@mkdir -p $(DIST)
	@new=$$({ find cmd internal -name '*.go' -exec shasum -a 256 {} + 2>/dev/null; \
	  find internal/game/assets -type f -exec shasum -a 256 {} + 2>/dev/null; \
	  shasum -a 256 go.mod go.sum 2>/dev/null; } | LC_ALL=C sort | shasum -a 256 | awk '{print $$1}'); \
	old=$$(cat $@ 2>/dev/null || true); \
	test "$$new" = "$$old" || echo "$$new" > $@

build: $(DIST)/$(BIN)

$(DIST)/$(BIN): $(INPUTS_SUM)
	mkdir -p $(DIST)
	go build -o $@ $(PKG)

wasm: $(WASM_DIR)/logos.zip

$(WASM_DIR)/logos.zip: $(WASM_DIR)/game.wasm $(WASM_DIR)/game-jam.wasm $(WASM_DIR)/index.html $(WASM_DIR)/wasm_exec.js
	cd $(WASM_DIR) && zip -q -FS logos.zip index.html wasm_exec.js game.wasm game-jam.wasm

$(WASM_DIR)/game.wasm: $(INPUTS_SUM)
	mkdir -p $(WASM_DIR)
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="$(DIST_LDFLAGS)" -o $@ $(PKG)

$(WASM_DIR)/game-jam.wasm: $(JAM_STAMP)

$(JAM_STAMP):
	mkdir -p $(WASM_DIR)
	@tmp=$$(mktemp -d) && \
		git -C "$(CURDIR)" archive "$(JAM_COMMIT)" | tar -xC "$$tmp" && \
		cd "$$tmp" && GOOS=js GOARCH=wasm go build -trimpath -ldflags="$(DIST_LDFLAGS)" -o "$(abspath $(WASM_DIR)/game-jam.wasm)" $(PKG) && \
		rm -rf "$$tmp"
	@touch $@

$(WASM_DIR)/wasm_exec.js: $(WASM_EXEC)
	mkdir -p $(WASM_DIR)
	cp $< $@

$(WASM_DIR)/index.html: wasm/index.html
	mkdir -p $(WASM_DIR)
	cp $< $@

serve-wasm: wasm
	cd $(WASM_DIR) && python3 -m http.server 8080

# Windows (amd64) cross-compile. Zipped from inside $(WIN_DIR) so the archive
# contains logos.exe at its root (itch.io players extract into their own folder).
build-win: $(WIN_DIR)/$(BIN).exe
	cd $(WIN_DIR) && zip -q -FS ../logos-windows-amd64.zip $(BIN).exe

$(WIN_DIR)/$(BIN).exe: $(INPUTS_SUM)
	mkdir -p $(WIN_DIR)
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$(WIN_LDFLAGS)" -o $@ $(PKG)

# Per-arch darwin binaries. Named with the arch suffix so they can coexist in
# the same directory alongside the fused universal binary produced by build-mac.
build-mac-arm64: $(MAC_DIR)/$(BIN)-arm64

$(MAC_DIR)/$(BIN)-arm64: $(INPUTS_SUM)
	mkdir -p $(MAC_DIR)
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$(DIST_LDFLAGS)" -o $@ $(PKG)

build-mac-amd64: $(MAC_DIR)/$(BIN)-amd64

$(MAC_DIR)/$(BIN)-amd64: $(INPUTS_SUM)
	mkdir -p $(MAC_DIR)
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="$(DIST_LDFLAGS)" -o $@ $(PKG)

# Fuses both mac binaries into a single universal (fat) Mach-O. The resulting
# $(BIN) file runs natively on Apple Silicon and Intel Macs. lipo is part of
# Xcode Command Line Tools — already installed wherever `go build` works on Mac.
build-mac: build-mac-arm64 build-mac-amd64
	lipo -create -output $(MAC_DIR)/$(BIN) $(MAC_DIR)/$(BIN)-arm64 $(MAC_DIR)/$(BIN)-amd64
	cd $(MAC_DIR) && zip -q -FS ../logos-mac-universal.zip $(BIN)

# One-shot target that produces everything itch.io needs:
#   dist/wasm/logos.zip                — browser build
#   dist/logos-windows-amd64.zip       — Windows 64-bit exe
#   dist/logos-mac-universal.zip       — macOS universal binary
dist: wasm build-win build-mac

clean:
	rm -rf $(DIST)/

FORCE:
