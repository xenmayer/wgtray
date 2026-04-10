.PHONY: build bundle clean

APP = WGTray.app
BIN = wgtray

build:
	go build -ldflags="-s -w" -o $(BIN) .

bundle: build
	mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Resources
	cp $(BIN) $(APP)/Contents/MacOS/
	cp icon/wgtray.icns $(APP)/Contents/Resources/AppIcon.icns
	cp Info.plist $(APP)/Contents/

install: bundle
	cp -r $(APP) /Applications/

clean:
	rm -rf $(BIN) $(APP)
