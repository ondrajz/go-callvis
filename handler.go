package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func analysisSetup() (r renderOpts) {
	r = renderOpts{
		focus:   *focusFlag,
		group:   []string{*groupFlag},
		ignore:  []string{*ignoreFlag},
		include: []string{*includeFlag},
		limit:   []string{*limitFlag},
		nointer: *nointerFlag,
		nostd:   *nostdFlag}

	return r
}

func processListArgs(r *renderOpts) (e error) {
	var groupBy []string
	for _, g := range strings.Split(r.group[0], ",") {
		g := strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if g != "pkg" && g != "type" {
			e = errors.New("invalid group option")
			return
		}
		groupBy = append(groupBy, g)
	}
	r.group = groupBy

	var ignorePaths []string
	for _, p := range strings.Split(r.ignore[0], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			ignorePaths = append(ignorePaths, p)
		}
	}
	r.ignore = ignorePaths

	var includePaths []string
	for _, p := range strings.Split(r.include[0], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			includePaths = append(includePaths, p)
		}
	}
	r.include = includePaths

	var limitPaths []string
	for _, p := range strings.Split(r.limit[0], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			limitPaths = append(limitPaths, p)
		}
	}
	r.limit = limitPaths

	return
}

func outputDot(fname string, outputPng bool) {
	// get cmdline default for analysis
	opts := analysisSetup()
	if e := processListArgs(&opts); e != nil {
		log.Fatalf("%v\n", e)
	}

	output, err := Analysis.render(opts)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	log.Println("writing dot output..")
	writeErr := ioutil.WriteFile(fmt.Sprintf("%s.gv", fname), output, 0755)
	if writeErr != nil {
		log.Fatalf("%v\n", writeErr)
	}

	if outputPng {
		log.Println("converting dot to svg..")
		_, err := dotToImage(fmt.Sprintf("%s.gv", fname), output)
		if err != nil {
			log.Fatalf("%v\n", err)
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && !strings.HasSuffix(r.URL.Path, ".svg") {
		http.NotFound(w, r)
		return
	}

	logf("----------------------")
	logf(" => handling request:  %v", r.URL)
	logf("----------------------")

	// get cmdline default for analysis
	opts := analysisSetup()

	// .. and allow overriding by HTTP params
	if f := r.FormValue("f"); f == "all" {
		opts.focus = ""
	} else if f != "" {
		opts.focus = f
	}
	if std := r.FormValue("std"); std != "" {
		opts.nostd = false
	}
	if inter := r.FormValue("nointer"); inter != "" {
		opts.nointer = true
	}
	if g := r.FormValue("group"); g != "" {
		opts.group[0] = g
	}
	if l := r.FormValue("limit"); l != "" {
		opts.limit[0] = l
	}
	if ign := r.FormValue("ignore"); ign != "" {
		opts.ignore[0] = ign
	}
	if inc := r.FormValue("include"); inc != "" {
		opts.include[0] = inc
	}

	// Convert list-style args to []string
	if e := processListArgs(&opts); e != nil {
		http.Error(w, "invalid group option", http.StatusInternalServerError)
		return
	}

	output, err := Analysis.render(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Form.Get("format") == "dot" {
		log.Println("writing dot output..")
		fmt.Fprint(w, string(output))
		return
	}

	log.Println("converting dot to svg..")
	img, err := dotToImage("", output)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("serving file:", img)
	http.ServeFile(w, r, img)
}
