APP     := WGTray
BINARY  := wgtray
DIST    := dist
APPDIR  := $(DIST)/$(APP).app
MACOS   := $(APPDIR)/Contents/MacOS
RES     := $(APPDIR)/Contents/Resources

.PHONY: all build bundle install clean build-linux

all: bundle

build:
	mkdir -p $(DIST)
	go build -o $(DIST)/$(BINARY) .

bundle: build
	mkdir -p $(MACOS) $(RES)
	cp $(DIST)/$(BINARY) $(MACOS)/$(BINARY)
	cp icon/wgtray.icns $(RES)/AppIcon.icns
	@echo '<?xml version="1.0" encoding="UTF-8"?>' > $(APPDIR)/Contents/Info.plist
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> $(APPDIR)/Contents/Info.plist
	@echo '<plist version="1.0"><dict>' >> $(APPDIR)/Contents/Info.plist
	@echo '  <key>CFBundleName</key><string>WGTray</string>' >> $(APPDIR)/Contents/Info.plist
	@echo '  <key>CFBundleIdentifier</key><string>com.xenmayer.wgtray</string>' >> $(APPDIR)/Contents/Info.plist
	@echo '  <key>CFBundleVersion</key><string>1.0</string>' >> $(APPDIR)/Contents/Info.plist
	@echo '  <key>CFBundleExecutable</key><string>wgtray</string>' >> $(APPDIR)/Contents/Info.plist
	@echo '  <key>CFBundleIconFile</key><string>AppIcon</string>' >> $(APPDIR)/Contents/Info.plist
	@echo '  <key>NSPrincipalClass</key><string>NSApplication</string>' >> $(APPDIR)/Contents/Info.plist
	@echo '  <key>LSUIElement</key><true/>' >> $(APPDIR)/Contents/Info.plist
	@echo '  <key>NSHighResolutionCapable</key><true/>' >> $(APPDIR)/Contents/Info.plist
	@echo '</dict></plist>' >> $(APPDIR)/Contents/Info.plist
	@echo "Built $(APPDIR)"

install: bundle
	rm -rf /Applications/$(APP).app
	cp -r $(APPDIR) /Applications/

clean:
	rm -rf $(DIST)

build-linux:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
	  go build -o $(DIST)/$(BINARY)-linux-amd64 .
