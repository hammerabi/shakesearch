package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"index/suffixarray"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	searcher := Searcher{}
	err := searcher.Load("justtitles.txt")
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/search", handleSearch(searcher))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fmt.Printf("Listening on port %s...", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

type WorksContent struct {
	Works map[string][]string
}

type Searcher struct {
	CompleteWorks string // Giant string of the entirety of shakespeare
	SuffixArray   *suffixarray.Index
	Scanner       *bufio.Scanner
	WorksMap      map[string][]string
}

func handleSearch(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, ok := r.URL.Query()["q"]
		if !ok || len(query[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search query in URL params"))
			return
		}
		results := searcher.Search(query[0])
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		err := enc.Encode(results)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("encoding failure"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf.Bytes())
	}
}

func (s *Searcher) Load(filename string) error {
	file, err := os.Open("justtitles.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	s.Scanner = scanner
	s.WorksMap = make(map[string][]string)
	s.GenerateWorksArray()

	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	s.CompleteWorks = string(dat)
	s.SuffixArray = suffixarray.New(dat)
	return nil
}

func (s *Searcher) GenerateWorksArray() {
	inContents := false // Flag that determines if we're currently in the table of contents
	for s.Scanner.Scan() {
		currentLine := strings.Trim(s.Scanner.Text(), " ")
		if currentLine == "Contents" {
			inContents = true
		}
		_, exists := s.WorksMap[currentLine]
		if exists && currentLine != "\n" && currentLine != "" {
			inContents = false
		}
		if inContents && currentLine != "\n" {
			var strArray []string
			s.WorksMap[currentLine] = strArray
		}
	}
	fmt.Printf("%+v", s.WorksMap)
	if err := s.Scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func (s *Searcher) Search(query string) []string {
	idxs := s.SuffixArray.Lookup([]byte(query), -1)
	results := []string{}
	for _, idx := range idxs {
		results = append(results, s.CompleteWorks[idx-250:idx+250])
	}
	return results
}
