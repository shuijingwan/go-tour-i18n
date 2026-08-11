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
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

// productionLocale is set by the release build. It is intentionally not a
// runtime flag: one production binary serves one locale.
var productionLocale = "zh-CN"

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("tour-production", flag.ContinueOnError)
	httpAddr := fs.String("http", "127.0.0.1:3999", "host:port to listen on")
	contentDir := fs.String("content", "", "published _content directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if strings.TrimSpace(*contentDir) == "" {
		return fmt.Errorf("--content is required")
	}
	handler, err := tour.NewProductionHandler(os.DirFS(*contentDir), productionLocale)
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
