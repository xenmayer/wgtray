.PHONY: build build-amd64 build-arm64 build-universal \
        bundle bundle-universal bundle-intel bundle-arm \
        dmg dmg-intel dmg-arm release install clean

APP        = WGTray.app
APP_INTEL  = WGTray-intel.app
APP_ARM    = WGTray-arm.app
BIN        = wgtray
BIN_INTEL  = wgtray-amd64
BIN_ARM    = wgtray-arm64
DMG        = dist/WGTray.dmg
DMG_INTEL  = dist/WGTray-intel.dmg
DMG_ARM    = dist/WGTray-arm.dmg

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    = -ldflags="-s -w"

# ─── Build ─────────────────────────────────────────────────────────────────────

build:
	go build $(LDFLAGS) -o $(BIN) .

# CGo cross-compilation: -arch tells Apple clang which target to emit.
build-amd64:
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
	CGO_CFLAGS="-arch x86_64 -mmacosx-version-min=10.15" \
	CGO_LDFLAGS="-arch x86_64 -mmacosx-version-min=10.15" \
	go build $(LDFLAGS) -o $(BIN_INTEL) .

build-arm64:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
	CGO_CFLAGS="-arch arm64 -mmacosx-version-min=11.0" \
	CGO_LDFLAGS="-arch arm64 -mmacosx-version-min=11.0" \
	go build $(LDFLAGS) -o $(BIN_ARM) .

build-universal: build-amd64 build-arm64
	lipo -create -output $(BIN) $(BIN_INTEL) $(BIN_ARM)
	@lipo -info $(BIN)

# ─── Bundle ────────────────────────────────────────────────────────────────────

bundle: build
	rm -rf $(APP)
	mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Resources
	cp $(BIN) $(APP)/Contents/MacOS/wgtray
	cp icon/wgtray.icns $(APP)/Contents/Resources/AppIcon.icns
	cp Info.plist $(APP)/Contents/
	codesign --sign - --force --deep $(APP)

bundle-universal: build-universal
	rm -rf $(APP)
	mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Resources
	cp $(BIN) $(APP)/Contents/MacOS/wgtray
	cp icon/wgtray.icns $(APP)/Contents/Resources/AppIcon.icns
	cp Info.plist $(APP)/Contents/
	codesign --sign - --force --deep $(APP)

bundle-intel: build-amd64
	rm -rf $(APP_INTEL)
	mkdir -p $(APP_INTEL)/Contents/MacOS $(APP_INTEL)/Contents/Resources
	cp $(BIN_INTEL) $(APP_INTEL)/Contents/MacOS/wgtray
	cp icon/wgtray.icns $(APP_INTEL)/Contents/Resources/AppIcon.icns
	cp Info.plist $(APP_INTEL)/Contents/
	codesign --sign - --force --deep $(APP_INTEL)

bundle-arm: build-arm64
	rm -rf $(APP_ARM)
	mkdir -p $(APP_ARM)/Contents/MacOS $(APP_ARM)/Contents/Resources
	cp $(BIN_ARM) $(APP_ARM)/Contents/MacOS/wgtray
	cp icon/wgtray.icns $(APP_ARM)/Contents/Resources/AppIcon.icns
	cp Info.plist $(APP_ARM)/Contents/
	codesign --sign - --force --deep $(APP_ARM)

# ─── DMG ───────────────────────────────────────────────────────────────────────

# Universal Binary DMG — one file for both Intel and Apple Silicon (primary artifact).
dmg: bundle-universal
	mkdir -p dist
	bash scripts/make-dmg.sh "$(APP)" "$(DMG)" "WGTray"

# Intel-only DMG (smaller download for x86_64 users).
dmg-intel: bundle-intel
	mkdir -p dist
	bash scripts/make-dmg.sh "$(APP_INTEL)" "$(DMG_INTEL)" "WGTray"

# Apple Silicon-only DMG (smaller download for arm64 users).
dmg-arm: bundle-arm
	mkdir -p dist
	bash scripts/make-dmg.sh "$(APP_ARM)" "$(DMG_ARM)" "WGTray"

# Build all three DMGs and print a summary.
release: dmg dmg-intel dmg-arm
	@echo ""
	@echo "✅  Release artifacts ($(VERSION)):"
	@ls -lh dist/*.dmg

# ─── Misc ──────────────────────────────────────────────────────────────────────

install: bundle
	cp -r $(APP) /Applications/

clean:
	rm -rf $(BIN) $(BIN_INTEL) $(BIN_ARM) \
	       $(APP) $(APP_INTEL) $(APP_ARM) \
	       dist/
