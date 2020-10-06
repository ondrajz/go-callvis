package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func analysisSetup() (r renderOpts) {
	r = renderOpts{
		cacheDir: *cacheDir,
		focus:    *focusFlag,
		group:    []string{*groupFlag},
		ignore:   []string{*ignoreFlag},
		include:  []string{*includeFlag},
		limit:    []string{*limitFlag},
		nointer:  *nointerFlag,
		nostd:    *nostdFlag}

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

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && !strings.HasSuffix(r.URL.Path, ".svg") {
		http.NotFound(w, r)
		return
	}

	logf("----------------------")
	logf(" => handling request:  %v", r.URL)
	logf("----------------------")

	opts := buildOptionsFromRequest(r)

	var img string
	if img = findCachedImg(opts); img != "" {
		log.Println("serving file:", img)
		http.ServeFile(w, r, img)
		return
	}

	// Convert list-style args to []string
	if e := processListArgs(opts); e != nil {
		http.Error(w, "invalid parameters", http.StatusBadRequest)
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

	log.Printf("converting dot to %s..\n", *outputFormat)

	img, err = dotToImage("", *outputFormat, output)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = cacheImg(opts, img)
	if err != nil {
		http.Error(w, "cache img error: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("serving file:", img)
	http.ServeFile(w, r, img)
}

func buildOptionsFromRequest(r *http.Request) *renderOpts {
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
	if refresh := r.FormValue("refresh"); refresh != "" {
		opts.refresh = true
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

	return &opts
}

func findCachedImg(opts *renderOpts) string {
	if opts.cacheDir == "" || opts.refresh {
		return ""
	}

	focus := opts.focus
	if focus == "" {
		focus = "all"
	}
	focusFilePath := focus + "." + *outputFormat
	absFilePath := filepath.Join(opts.cacheDir, focusFilePath)

	if exists, err := pathExists(absFilePath); err != nil || !exists {
		log.Println("not cached img:", absFilePath)
		return ""
	}

	log.Println("hit cached img")
	return absFilePath
}

func cacheImg(opts *renderOpts, img string) error {
	if opts.cacheDir == "" || img == "" {
		return nil
	}

	focus := opts.focus
	if focus == "" {
		focus = "all"
	}
	absCacheDirPrefix := filepath.Join(opts.cacheDir, focus)
	absCacheDirPath := strings.TrimRightFunc(absCacheDirPrefix, func(r rune) bool {
		return r != '\\' && r != '/'
	})
	err := os.MkdirAll(absCacheDirPath, os.ModePerm)
	if err != nil {
		return err
	}

	absFilePath := absCacheDirPrefix + "." + *outputFormat
	_, err = copyFile(img, absFilePath)
	if err != nil {
		return err
	}

	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
