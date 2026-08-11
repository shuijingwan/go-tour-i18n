// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

// productionLocale is set by the release build. It is intentionally not a
// runtime flag: one production binary serves one locale.
var productionLocale string

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("tour-production", flag.ContinueOnError)
	httpAddr := fs.String("http", "127.0.0.1:3999", "host:port to listen on")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if strings.TrimSpace(productionLocale) == "" {
		return fmt.Errorf("production locale was not injected at build time")
	}
	contentDir, err := bundleContentDir()
	if err != nil {
		return err
	}
	handler, err := tour.NewProductionHandler(os.DirFS(contentDir), productionLocale)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              *httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	log.Printf("serving locale %s on http://%s with remote Playground execution", productionLocale, *httpAddr)
	return server.ListenAndServe()
}

// bundleContentDir resolves _content relative to this binary, rather than to
// the caller's current working directory.
func bundleContentDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate production binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	contentDir := filepath.Clean(filepath.Join(filepath.Dir(executable), "..", "_content"))
	info, err := os.Stat(contentDir)
	if err != nil {
		return "", fmt.Errorf("locate bundle content %q: %w", contentDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("bundle content %q is not a directory", contentDir)
	}
	return contentDir, nil
}
