// wotd - word of the day backend for a Noctalia widget.
//
// Usage:
//
//	wotd                          print today's entry as JSON (cached per day)
//	wotd -word es -meaning en     override languages for this call (ISO 639-1)
//	wotd -text                    print a human-readable version
//	wotd -refresh                 ignore the cache
//	wotd set -word es -meaning en save default languages (CLI use only; the
//	                              Noctalia plugin passes its own settings)
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

type Config struct {
	WordLang    string `json:"word_lang"`
	MeaningLang string `json:"meaning_lang"`
}

// Bump when the shape of Entry changes so stale cache files are ignored.
const entryVersion = 2

type Example struct {
	Text        string `json:"text"`
	Translation string `json:"translation,omitempty"`
}

type Entry struct {
	Version     int       `json:"version"`
	Date        string    `json:"date"`
	Word        string    `json:"word"`
	Meaning     string    `json:"meaning"`
	Examples    []Example `json:"examples,omitempty"`
	WordLang    string    `json:"word_lang"`
	MeaningLang string    `json:"meaning_lang"`
	Error       string    `json:"error,omitempty"`
}

var client = &http.Client{Timeout: 10 * time.Second}

func dir(base func() (string, error)) string {
	d, err := base()
	if err != nil {
		d = os.TempDir()
	}
	p := filepath.Join(d, "wotd")
	_ = os.MkdirAll(p, 0o755)
	return p
}

func configPath() string { return filepath.Join(dir(os.UserConfigDir), "config.json") }
func cacheDir() string   { return dir(os.UserCacheDir) }

func loadConfig() Config {
	c := Config{WordLang: "es", MeaningLang: "en"}
	if b, err := os.ReadFile(configPath()); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	return c
}

func saveConfig(c Config) error {
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(configPath(), b, 0o644)
}

func get(u string) ([]byte, error) {
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "wotd/0.1 (personal vocabulary widget)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET %s: %s", u, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// wordList downloads (once) a frequency list for the language and keeps the
// top ~20k plain words, so you learn common vocabulary first.
func wordList(lang string) ([]string, error) {
	path := filepath.Join(cacheDir(), "words_"+lang+".txt")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		u := fmt.Sprintf("https://raw.githubusercontent.com/hermitdave/FrequencyWords/master/content/2018/%[1]s/%[1]s_50k.txt", lang)
		b, err := get(u)
		if err != nil {
			return nil, fmt.Errorf("no word list for %q: %w", lang, err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return nil, err
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var words []string
	sc := bufio.NewScanner(f)
	for sc.Scan() && len(words) < 20000 {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		w := fields[0]
		if len([]rune(w)) >= 4 && strings.IndexFunc(w, func(r rune) bool { return !unicode.IsLetter(r) }) < 0 {
			words = append(words, w)
		}
	}
	if len(words) == 0 {
		return nil, errors.New("word list is empty")
	}
	return words, nil
}

// Same word for the whole day, different each day.
func pick(words []string, date, lang string) string {
	h := fnv.New32a()
	h.Write([]byte(date + lang))
	return words[int(h.Sum32())%len(words)]
}

// Different languages: translate with MyMemory (free, no key, rate limited).
func translate(word, from, to string) (string, error) {
	u := fmt.Sprintf("https://api.mymemory.translated.net/get?q=%s&langpair=%s|%s",
		url.QueryEscape(word), from, to)
	b, err := get(u)
	if err != nil {
		return "", err
	}
	var r struct {
		ResponseData struct {
			TranslatedText string `json:"translatedText"`
		} `json:"responseData"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}
	if r.ResponseData.TranslatedText == "" {
		return "", errors.New("empty translation")
	}
	return r.ResponseData.TranslatedText, nil
}

// Same language: pull a definition from that language's Wiktionary (best effort).
func define(word, lang string) (string, error) {
	u := fmt.Sprintf("https://%s.wiktionary.org/w/api.php?action=query&prop=extracts&explaintext=1&redirects=1&format=json&titles=%s",
		lang, url.QueryEscape(word))
	b, err := get(u)
	if err != nil {
		return "", err
	}
	var r struct {
		Query struct {
			Pages map[string]struct {
				Extract string `json:"extract"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}
	for _, p := range r.Query.Pages {
		var keep []string
		for _, line := range strings.Split(p.Extract, "\n") {
			line = strings.TrimSpace(line)
			// skip empty lines and section headings like "== Noun =="
			if line == "" || strings.HasPrefix(line, "=") {
				continue
			}
			keep = append(keep, line)
			if len(keep) == 3 {
				break
			}
		}
		if len(keep) > 0 {
			return strings.Join(keep, " · "), nil
		}
	}
	return "", errors.New("no definition found")
}

// Overridable so tests can point at a fake server.
var tatoebaURL = "https://tatoeba.org/en/api_v0/search"

// Tatoeba wants ISO 639-3 codes.
var lang3 = map[string]string{
	"en": "eng", "es": "spa", "fr": "fra", "de": "deu", "it": "ita", "pt": "por",
	"nl": "nld", "sv": "swe", "pl": "pol", "ru": "rus", "tr": "tur", "ro": "ron",
}

// examples fetches up to two example sentences from Tatoeba. When the meaning
// language differs, only sentences that have a translation into it are used.
// Failure here is never fatal: the entry just has no examples.
func examples(word, from, to string) ([]Example, error) {
	f, ok := lang3[from]
	if !ok {
		return nil, fmt.Errorf("no Tatoeba code for %q", from)
	}
	q := url.Values{}
	q.Set("from", f)
	q.Set("query", word)
	t := ""
	if from != to {
		t, ok = lang3[to]
		if !ok {
			return nil, fmt.Errorf("no Tatoeba code for %q", to)
		}
		q.Set("to", t)
		q.Set("trans_filter", "limit")
	}
	b, err := get(tatoebaURL + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	var r struct {
		Results []struct {
			Text         string `json:"text"`
			Translations [][]struct {
				Text string `json:"text"`
				Lang string `json:"lang"`
			} `json:"translations"`
		} `json:"results"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}

	var good, rest []Example
	for _, res := range r.Results {
		ex := Example{Text: res.Text}
		if t != "" {
			for _, group := range res.Translations {
				for _, tr := range group {
					if tr.Lang == t && ex.Translation == "" {
						ex.Translation = tr.Text
					}
				}
			}
			if ex.Translation == "" {
				continue
			}
		}
		// Prefer medium-length sentences: short ones are trivial, long ones don't fit.
		if n := len([]rune(ex.Text)); n >= 20 && n <= 100 {
			good = append(good, ex)
		} else {
			rest = append(rest, ex)
		}
	}
	out := append(good, rest...)
	if len(out) > 2 {
		out = out[:2]
	}
	return out, nil
}

func today(cfg Config, refresh bool) Entry {
	date := time.Now().Format("2006-01-02")
	cache := filepath.Join(cacheDir(), fmt.Sprintf("today_%s_%s.json", cfg.WordLang, cfg.MeaningLang))

	if !refresh {
		if b, err := os.ReadFile(cache); err == nil {
			var e Entry
			if json.Unmarshal(b, &e) == nil && e.Date == date &&
				e.WordLang == cfg.WordLang && e.MeaningLang == cfg.MeaningLang &&
				e.Version == entryVersion && e.Error == "" {
				return e
			}
		}
	}

	e := Entry{Version: entryVersion, Date: date, WordLang: cfg.WordLang, MeaningLang: cfg.MeaningLang}
	words, err := wordList(cfg.WordLang)
	if err != nil {
		e.Error = err.Error()
		return e
	}
	e.Word = pick(words, date, cfg.WordLang)

	if cfg.WordLang == cfg.MeaningLang {
		e.Meaning, err = define(e.Word, cfg.WordLang)
	} else {
		e.Meaning, err = translate(e.Word, cfg.WordLang, cfg.MeaningLang)
	}
	if err != nil {
		e.Error = err.Error()
		return e // not cached, so the next run retries
	}
	if ex, err := examples(e.Word, cfg.WordLang, cfg.MeaningLang); err == nil {
		e.Examples = ex
	}
	b, _ := json.Marshal(e)
	_ = os.WriteFile(cache, b, 0o644)
	return e
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "set" {
		fs := flag.NewFlagSet("set", flag.ExitOnError)
		cfg := loadConfig()
		fs.StringVar(&cfg.WordLang, "word", cfg.WordLang, "language of the word")
		fs.StringVar(&cfg.MeaningLang, "meaning", cfg.MeaningLang, "language of the meaning")
		_ = fs.Parse(os.Args[2:])
		if err := saveConfig(cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("word=%s meaning=%s\n", cfg.WordLang, cfg.MeaningLang)
		return
	}

	text := flag.Bool("text", false, "human-readable output")
	refresh := flag.Bool("refresh", false, "ignore today's cache")
	word := flag.String("word", "", "language of the word (overrides saved config)")
	meaning := flag.String("meaning", "", "language of the meaning (overrides saved config)")
	flag.Parse()

	cfg := loadConfig()
	if *word != "" {
		cfg.WordLang = *word
	}
	if *meaning != "" {
		cfg.MeaningLang = *meaning
	}
	e := today(cfg, *refresh)
	if *text {
		if e.Error != "" {
			fmt.Println("error:", e.Error)
			os.Exit(1)
		}
		fmt.Printf("%s – %s\n", e.Word, e.Meaning)
		for _, ex := range e.Examples {
			fmt.Printf("  • %s\n", ex.Text)
			if ex.Translation != "" {
				fmt.Printf("    %s\n", ex.Translation)
			}
		}
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(e)
}
