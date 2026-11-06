package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func structuredRecoveryCommand(root string, args []string) error {
	fs := flag.NewFlagSet("site-content structured-recover", flag.ContinueOnError)
	bundle := fs.String("bundle", "", "原始模型输入 Bundle，允许完整本机路径")
	input := fs.String("input", "", "原始 Generation JSON，允许完整本机路径")
	bundleSHA := fs.String("bundle-sha256", "", "已确认的原始 Bundle SHA-256")
	outputSHA := fs.String("output-sha256", "", "已确认的原始 output SHA-256")
	provider := fs.String("provider", "", "原始 Generation provider")
	model := fs.String("model", "", "原始 Generation model")
	session := fs.String("session", "", "原始 Generation session")
	at := fs.String("generated-at", "", "原始 Generation 时间，RFC3339 UTC")
	apply := fs.Bool("apply", false, "保存原始 bytes 与 immutable carry-forward evidence；默认只读 preflight")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *bundle == "" || *input == "" || *bundleSHA == "" || *outputSHA == "" || *provider == "" || *model == "" || *session == "" || *at == "" {
		return fmt.Errorf("structured-recover 必须提供原始 bundle/input、两个 SHA 与完整原始 provider/model/session/generated-at，不接受额外位置参数")
	}
	read := func(p string) ([]byte, error) {
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		st, err := os.Lstat(p)
		if err != nil {
			return nil, err
		}
		if !st.Mode().IsRegular() || st.Size() > 32<<20 {
			return nil, fmt.Errorf("recovery input must be a regular bounded file: %s", p)
		}
		return os.ReadFile(p)
	}
	b, err := read(*bundle)
	if err != nil {
		return err
	}
	o, err := read(*input)
	if err != nil {
		return err
	}
	r, err := sitecontent.RecoverStructuredAssets(root, b, o, sitecontent.StructuredRecoveryOptions{BundleSHA: *bundleSHA, OutputSHA: *outputSHA, Provenance: sitecontent.Provenance{Provider: *provider, Model: *model, Session: *session, GeneratedAt: *at}, Apply: *apply})
	if err != nil {
		return err
	}
	return printJSON(r)
}
