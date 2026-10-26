# wotd - a dictionary widget for Noctalia v5, with a Go backend

A learning/experiment project. Pick a language for the word and a language for
the meaning; the widget shows an entry like a small dictionary:

    critiques – critics
    • Les critiques ont aimé le film.
       The critics liked the movie.

## Layout

    wotd/
    ├── main.go            Go helper: word choice, meaning, examples, caching
    ├── main_test.go       tests (fake Tatoeba server, deterministic pick)
    ├── Makefile           build / test / export-to-plugins shortcuts
    └── plugin/            the Noctalia plugin that gets copied to the plugins dir
        ├── plugin.toml        manifest + settings (language pickers)
        ├── widget.luau        bar widget
        ├── desktop.luau       desktop widget (the dictionary tile)
        ├── lib/core.luau      shared: runs the Go helper, parses its JSON
        └── translations/en.json

## Workflow

    make test         # go vet + tests
    make run          # print today's entry in the terminal, fresh from the network
    make install      # build -> ~/.local/bin/wotd, copy plugin -> Noctalia plugins dir
    make enable       # first time only
    make reload       # after changing plugin.toml (disable + enable)

Edits to `.luau` files hot-reload after `make plugin`. Edits to Go need
`make bin` (the plugin runs the installed binary, not `go run`).
Languages are chosen in Noctalia: Settings -> Plugins -> gear icon.

## How data flows

    Noctalia (Luau)  --runAsync argv-->  wotd (Go)  --HTTP-->  word list (GitHub)
    widget/desktop        JSON on stdout                       MyMemory / Wiktionary (meaning)
                                                               Tatoeba (examples)

The helper caches one JSON file per language pair per day in ~/.cache/wotd/.

## Ideas to experiment with

- Lemmas: frequency lists contain inflected forms ("descanse" = "rest!"). Skip
  words that have no dictionary headword, or use a lemmatised list.
- Part of speech / IPA / gender via Wiktionary or Kaikki.org dumps (offline!).
- Word history: append each day's entry to a file; add a panel to browse it.
- Spaced repetition: re-show words you got wrong (needs a panel with buttons).
- Swap translation providers (LibreTranslate, DeepL) behind a small interface.
- More languages: add to lang3 (main.go) and the options in plugin/plugin.toml.
- Replace the 1-hour poll with a timer aimed at midnight.

## Known unknowns

Written against the Noctalia v5 plugin docs; the Tatoeba response shape and
the exact desktop-widget styling were not verified against the live services.
