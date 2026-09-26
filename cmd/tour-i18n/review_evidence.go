package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func reviewEvidenceCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: review-evidence <check --file <path>|save --input <draft.md> --output <formal.md>>")
	}
	switch args[0] {
	case "check":
		fs := flag.NewFlagSet("review-evidence check", flag.ContinueOnError)
		path := fs.String("file", "", "Reviewer Markdown evidence file")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if strings.TrimSpace(*path) == "" || fs.NArg() != 0 {
			return fmt.Errorf("usage: review-evidence check --file <path>")
		}
		if err := checkReviewEvidenceFile(*path); err != nil {
			return err
		}
		fmt.Printf("Reviewer Markdown evidence format: PASS (file=%s)\n", *path)
		return nil
	case "save":
		fs := flag.NewFlagSet("review-evidence save", flag.ContinueOnError)
		input := fs.String("input", "", "Reviewer Markdown draft")
		output := fs.String("output", "", "new formal Reviewer Markdown evidence")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if strings.TrimSpace(*input) == "" || strings.TrimSpace(*output) == "" || fs.NArg() != 0 {
			return fmt.Errorf("usage: review-evidence save --input <draft.md> --output <formal.md>")
		}
		path, err := saveReviewEvidence(*input, *output)
		if err != nil {
			return err
		}
		fmt.Printf("Reviewer Markdown evidence saved: %s\n", path)
		return nil
	default:
		return fmt.Errorf("usage: review-evidence <check --file <path>|save --input <draft.md> --output <formal.md>>")
	}
}

func checkReviewEvidenceFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Reviewer Markdown evidence %q: %w", path, err)
	}
	return validateReviewEvidence(path, data)
}

func validateReviewEvidence(path string, data []byte) error {
	formatError := func(line int, reason string) error {
		return fmt.Errorf("%s: line %d: %s", path, line, reason)
	}
	if len(bytes.Trim(data, " \t\r\n")) == 0 {
		return formatError(1, "file is empty")
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return formatError(1, "UTF-8 BOM is not allowed")
	}
	if !utf8.Valid(data) {
		for offset := 0; offset < len(data); {
			_, size := utf8.DecodeRune(data[offset:])
			if size == 1 && data[offset] >= utf8.RuneSelf {
				line := bytes.Count(data[:offset], []byte{'\n'}) + 1
				return formatError(line, "invalid UTF-8")
			}
			offset += size
		}
		return formatError(1, "invalid UTF-8")
	}
	for offset, value := range data {
		if value != '\r' {
			continue
		}
		line := bytes.Count(data[:offset], []byte{'\n'}) + 1
		if offset+1 < len(data) && data[offset+1] == '\n' {
			return formatError(line, "CRLF newline is not allowed; use LF")
		}
		return formatError(line, "carriage return is not allowed; use LF newlines")
	}
	lineStart := 0
	lineNumber := 1
	for lineStart < len(data) {
		lineEnd := bytes.IndexByte(data[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(data)
		} else {
			lineEnd += lineStart
		}
		if lineEnd > lineStart && (data[lineEnd-1] == ' ' || data[lineEnd-1] == '\t') {
			return formatError(lineNumber, "trailing space or tab is not allowed")
		}
		if lineEnd == len(data) {
			break
		}
		lineStart = lineEnd + 1
		lineNumber++
	}
	if data[len(data)-1] != '\n' {
		return formatError(bytes.Count(data, []byte{'\n'})+1, "file must end with exactly one LF")
	}
	if len(data) >= 2 && data[len(data)-2] == '\n' {
		return formatError(bytes.Count(data[:len(data)-1], []byte{'\n'})+1, "extra blank line at end of file")
	}
	return nil
}

func saveReviewEvidence(input, output string) (path string, err error) {
	data, err := os.ReadFile(input)
	if err != nil {
		return "", fmt.Errorf("read Reviewer Markdown draft %q: %w", input, err)
	}
	if err := validateReviewEvidence(input, data); err != nil {
		return "", err
	}
	path, err = filepath.Abs(output)
	if err != nil {
		return "", fmt.Errorf("resolve Reviewer Markdown evidence output: %w", err)
	}
	if _, err := os.Lstat(path); err == nil {
		return "", fmt.Errorf("Reviewer Markdown evidence already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect Reviewer Markdown evidence output: %w", err)
	}
	parent := filepath.Dir(path)
	info, err := os.Stat(parent)
	if err != nil {
		return "", fmt.Errorf("inspect Reviewer Markdown evidence output directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("Reviewer Markdown evidence output parent %q is not a directory", parent)
	}
	temporary, err := os.CreateTemp(parent, "."+filepath.Base(path)+".review-evidence-")
	if err != nil {
		return "", fmt.Errorf("create Reviewer Markdown evidence staging file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err = temporary.Chmod(0644); err != nil {
		return "", fmt.Errorf("set Reviewer Markdown evidence staging mode: %w", err)
	}
	if _, err = temporary.Write(data); err != nil {
		return "", fmt.Errorf("write Reviewer Markdown evidence staging file: %w", err)
	}
	if err = temporary.Sync(); err != nil {
		return "", fmt.Errorf("sync Reviewer Markdown evidence staging file: %w", err)
	}
	if err = temporary.Close(); err != nil {
		return "", fmt.Errorf("close Reviewer Markdown evidence staging file: %w", err)
	}
	if err = os.Link(temporaryPath, path); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("Reviewer Markdown evidence already exists: %s", path)
		}
		return "", fmt.Errorf("install Reviewer Markdown evidence: %w", err)
	}
	return path, nil
}
