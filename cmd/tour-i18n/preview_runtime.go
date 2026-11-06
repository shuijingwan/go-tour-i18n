package main

import (
	"net/http"
	"os"

	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

func newFullLocalePreviewHandler(root, contentDir, locale string) (http.Handler, error) {
	tourHandler, err := tour.NewPreviewHandler(os.DirFS(contentDir), locale)
	if err != nil {
		return nil, err
	}
	origin, err := sitecontent.LocaleOrigin(root, locale)
	if err != nil {
		return nil, err
	}
	content, err := site.LoadContent(root, locale, origin)
	if err != nil {
		return nil, err
	}
	return content.Compose(locale, tourHandler)
}
