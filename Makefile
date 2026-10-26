# wotd - dictionary widget for Noctalia v5
# Run `make help` to see the targets.

PLUGIN_ID  := me/word-of-the-day
PLUGIN_DST := $(or $(XDG_DATA_HOME),$(HOME)/.local/share)/noctalia/plugins/word-of-the-day
BIN_DST    := $(HOME)/.local/bin/wotd

.PHONY: help build test run bin plugin install enable disable reload

help:      ## list targets
	@grep -E '^[a-z]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*##/\t/'

build:     ## compile the Go helper into ./bin/wotd
	go build -o bin/wotd .

test:      ## vet + unit tests
	go vet ./... && go test ./...

run:       ## fetch today's entry (ignores cache) and print it
	go run . -refresh -text

bin: build ## install the helper to ~/.local/bin/wotd
	install -Dm755 bin/wotd $(BIN_DST)

plugin:    ## copy ./plugin into Noctalia's plugin dir (.luau edits hot-reload)
	mkdir -p $(PLUGIN_DST)
	cp -r plugin/. $(PLUGIN_DST)/
	@echo "exported to $(PLUGIN_DST)"

install: bin plugin ## helper + plugin

enable:    ## enable the plugin in Noctalia
	noctalia msg plugins enable $(PLUGIN_ID)

disable:   ## disable the plugin
	noctalia msg plugins disable $(PLUGIN_ID)

reload: plugin ## re-export, then disable+enable (needed after plugin.toml changes)
	-noctalia msg plugins disable $(PLUGIN_ID)
	noctalia msg plugins enable $(PLUGIN_ID)
