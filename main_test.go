package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The JSON below is the shape I *believe* Tatoeba returns (translations is a
// list of lists). If the real API differs, this test still passes but real
// examples will be empty - check with: go run . -refresh -text
const fakeTatoeba = `{"results":[
 {"text":"Oui.","translations":[[{"text":"Yes.","lang":"eng"}]]},
 {"text":"Les critiques ont aimé le film.","translations":[[],[{"text":"The critics liked the movie.","lang":"eng"}]]},
 {"text":"Il n'a pas de traduction anglaise ici, mais assez long.","translations":[[{"text":"Es hat","lang":"deu"}]]},
 {"text":"Les critiques sont parfois très sévères avec lui.","translations":[[{"text":"Critics are sometimes harsh on him.","lang":"eng"}]]}
]}`

func TestExamplesTranslated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("from") != "fra" || q.Get("to") != "eng" || q.Get("query") != "critiques" {
			t.Errorf("unexpected query: %v", q)
		}
		w.Write([]byte(fakeTatoeba))
	}))
	defer srv.Close()
	old := tatoebaURL
	tatoebaURL = srv.URL
	defer func() { tatoebaURL = old }()

	got, err := examples("critiques", "fr", "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 examples, got %d: %+v", len(got), got)
	}
	// medium-length sentences with an English translation win; "Oui." and the
	// German-only one are excluded/deprioritised.
	if got[0].Translation != "The critics liked the movie." || got[1].Translation != "Critics are sometimes harsh on him." {
		t.Errorf("unexpected examples: %+v", got)
	}
}

func TestExamplesSameLanguage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("to") != "" {
			t.Errorf("same-language lookup must not send 'to'")
		}
		w.Write([]byte(fakeTatoeba))
	}))
	defer srv.Close()
	old := tatoebaURL
	tatoebaURL = srv.URL
	defer func() { tatoebaURL = old }()

	got, err := examples("critiques", "fr", "fr")
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v err %v", got, err)
	}
	if got[0].Translation != "" {
		t.Errorf("same-language examples should have no translation: %+v", got[0])
	}
}

func TestPickIsStable(t *testing.T) {
	words := []string{"a", "b", "c", "d", "e"}
	if pick(words, "2026-10-26", "fr") != pick(words, "2026-10-26", "fr") {
		t.Error("pick must be deterministic for the same day")
	}
}
