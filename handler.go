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

	focus := *focusFlag
	nostd := *nostdFlag
	nointer := *nointerFlag
	group := *groupFlag
	limit := *limitFlag
	ignore := *ignoreFlag
	include := *includeFlag

	if f := r.FormValue("f"); f == "all" {
		focus = ""
	} else if f != "" {
		focus = f
	}
	if std := r.FormValue("std"); std != "" {
		nostd = false
	}
	if inter := r.FormValue("nointer"); inter != "" {
		nointer = true
	}
	if g := r.FormValue("group"); g != "" {
		group = g
	}
	if l := r.FormValue("limit"); l != "" {
		limit = l
	}
	if ign := r.FormValue("ignore"); ign != "" {
		ignore = ign
	}
	if inc := r.FormValue("include"); inc != "" {
		include = inc
	}

	var groupBy []string
	for _, g := range strings.Split(group, ",") {
		g := strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if g != "pkg" && g != "type" {
			http.Error(w, "invalid group option", http.StatusInternalServerError)
			return
		}
		groupBy = append(groupBy, g)
	}

	var ignorePaths []string
	for _, p := range strings.Split(ignore, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			ignorePaths = append(ignorePaths, p)
		}
	}

	var includePaths []string
	for _, p := range strings.Split(include, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			includePaths = append(includePaths, p)
		}
	}

	var limitPaths []string
	for _, p := range strings.Split(limit, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			limitPaths = append(limitPaths, p)
		}
	}

	opts := renderOpts{
		focus:   focus,
		group:   groupBy,
		ignore:  ignorePaths,
		include: includePaths,
		limit:   limitPaths,
		nointer: nointer,
		nostd:   nostd,
	}

	output, err := Analysis.render(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Form.Get("format") == "dot" {
		log.Println("writing dot output..")
		fmt.Fprint(w, output)
		return
	}

	log.Println("converting dot to svg..")
	img, err := dotToImage(output)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("serving file:", img)
	http.ServeFile(w, r, img)

}
