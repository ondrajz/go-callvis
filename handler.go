package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && !strings.HasSuffix(r.URL.Path, ".svg") {
		http.NotFound(w, r)
		return
	}

	logf("----------------------")
	logf(" => handling request:  %v", r.URL)
	logf("----------------------")

	// get cmdline default for analysis
	Analysis.OptsSetup()

	// .. and allow overriding by HTTP params
	Analysis.OverrideByHTTP(r)

	// Convert list-style args to []string
	if e := Analysis.processListArgs(); e != nil {
		http.Error(w, "invalid group option", http.StatusInternalServerError)
		return
	}

	output, err := Analysis.Render()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Form.Get("format") == "dot" {
		log.Println("writing dot output..")
		fmt.Fprint(w, string(output))
		return
	}

	log.Printf("converting dot to %s..\n", *outputFormat)

	img, err := dotToImage("", *outputFormat, output)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("serving file:", img)
	http.ServeFile(w, r, img)
}
