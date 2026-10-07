package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func TestPublicationPolicyCommandUsesTourPolicyAuthority(t *testing.T) {
	for _, test := range []struct {
		locale      string
		publication string
		advertising string
		ads         bool
	}{
		{locale: "zh-CN", publication: "go-local", advertising: "go-local", ads: false},
		{locale: "sw-TZ", publication: "standard", advertising: "ads-unsupported", ads: false},
		{locale: "kk-KZ", publication: "standard", advertising: "ads-unsupported", ads: false},
		{locale: "fa-IR", publication: "standard", advertising: "ads-unsupported", ads: false},
		{locale: "am-ET", publication: "standard", advertising: "ads-unsupported", ads: false},
		{locale: "ja-JP", publication: "standard", advertising: "standard", ads: true},
		{locale: "tr-TR", publication: "standard", advertising: "standard", ads: true},
	} {
		t.Run(test.locale, func(t *testing.T) {
			output := captureStdout(t, func() error {
				return publicationPolicyCommand([]string{"--locale", test.locale})
			})
			var got struct {
				Locale         string `json:"locale"`
				Publication    string `json:"publication"`
				Advertising    string `json:"advertising"`
				TourAdsEnabled bool   `json:"tour_ads_enabled"`
			}
			if err := json.Unmarshal(output, &got); err != nil {
				t.Fatal(err)
			}
			if got.Locale != test.locale || got.Publication != test.publication || got.Advertising != test.advertising || got.TourAdsEnabled != test.ads {
				t.Fatalf("publication output = %+v, want locale=%q publication=%q advertising=%q tour_ads_enabled=%t", got, test.locale, test.publication, test.advertising, test.ads)
			}
		})
	}
}

func TestPublicationPolicyCommandRejectsInvalidArguments(t *testing.T) {
	if err := publicationPolicyCommand(nil); err == nil {
		t.Fatal("publication policy command accepted missing locale")
	}
}

func captureStdout(t *testing.T, call func() error) []byte {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = original })
	if err := call(); err != nil {
		writer.Close()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(output)
}
