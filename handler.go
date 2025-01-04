package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && !strings.HasSuffix(r.URL.Path, ".svg") {
		http.NotFound(w, r)
		return
	}


	logf("----------------------")
	logf(" => handling request: %v %v", r.Method, r.URL)
	logf("----------------------")

	t0 := time.Now()
	defer func() {
		logf("----------------------")
		logf(" -> request done (took %.3fs):  %v", time.Since(t0).Round(time.Millisecond).Seconds(), r.URL)
		logf("----------------------")
	}()

	// set up cmdline default for analysis
	Analysis.OptsSetup()

	// .. and allow overriding by HTTP params
	Analysis.OverrideByHTTP(r)

	var img string
	if img = Analysis.FindCachedImg(); img != "" {
		log.Println("serving cached file:", img)
		http.ServeFile(w, r, img)
		return
	}

	// Convert list-style args to []string
	if e := Analysis.ProcessListArgs(); e != nil {
		http.Error(w, "invalid parameters", http.StatusBadRequest)
		return
	}

	output, err := Analysis.Render()
	if err != nil {
		http.Error(w, fmt.Sprintf("rendering failed: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	if r.Form.Get("format") == "dot" {
		log.Println("writing dot output")
		fmt.Fprint(w, string(output))
		return
	}

	log.Printf("converting dot to %s\n", *outputFormat)
	t := time.Now()
	
	img, err = dotToImage("", *outputFormat, output)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("conversion done (took %.3fs)", time.Since(t).Round(time.Millisecond).Seconds())

	err = Analysis.CacheImg(img)
	if err != nil {
		http.Error(w, "cache img error: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("serving file:", img)
	http.ServeFile(w, r, img)
}

