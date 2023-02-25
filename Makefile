APP_NAME=nas_checker_bot
APP_SRC=*.go
BUILD_DIR=build

.PHONY: build clean

build:
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME) $(APP_SRC)

clean:
	rm -rf $(BUILD_DIR)
