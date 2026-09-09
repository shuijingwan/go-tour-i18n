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
		locale string
		want   bool
	}{
		{locale: "zh-CN", want: false},
		{locale: "ja-JP", want: true},
	} {
		t.Run(test.locale, func(t *testing.T) {
			output := captureStdout(t, func() error {
				return publicationPolicyCommand([]string{"--locale", test.locale})
			})
			var got struct {
				Locale         string `json:"locale"`
				TourAdsEnabled bool   `json:"tour_ads_enabled"`
			}
			if err := json.Unmarshal(output, &got); err != nil {
				t.Fatal(err)
			}
			if got.Locale != test.locale || got.TourAdsEnabled != test.want {
				t.Fatalf("publication output = %+v, want locale=%q tour_ads_enabled=%t", got, test.locale, test.want)
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
