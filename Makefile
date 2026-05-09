BIN_NAME    ?= ciscobuddy
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: all build install clean run-example

all: build

build:
	go build -o $(BIN_NAME) .

install: build
	mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BIN_NAME) $(INSTALL_DIR)/$(BIN_NAME)
	@echo "Installed to $(INSTALL_DIR)/$(BIN_NAME)"

clean:
	rm -f $(BIN_NAME) $(BIN_NAME).exe

run-example: build
	./$(BIN_NAME) -ip 194.1.10.180 194.1.10.255 -r 2 \
	    -name "Network A" -dev 42 \
	    -name "Network B" -dev 2
