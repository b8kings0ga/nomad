// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package getter

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMinimalGetterHTTPFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "nomad-min")
	}))
	defer srv.Close()
	dir := t.TempDir()
	dst := filepath.Join(dir, "artifact")
	p := &parameters{Source: srv.URL + "/artifact", Destination: dst, AllocDir: dir, Mode: artifactModeFile, HTTPMaxBytes: 1024}
	if err := p.getMinimal(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "nomad-min" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestMinimalGetterRejectsScheme(t *testing.T) {
	dir := t.TempDir()
	p := &parameters{Source: "s3://bucket/key", Destination: filepath.Join(dir, "artifact"), AllocDir: dir}
	err := p.getMinimal(context.Background())
	if err == nil || !strings.Contains(err.Error(), "nomad_min: unsupported feature artifact scheme") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMinimalGetterRejectsArchiveEscape(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "bad.tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "../escape", Mode: 0644, Size: 1}); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	_ = gz.Close()
	_ = f.Close()
	p := &parameters{Source: archive, Destination: filepath.Join(dir, "out"), AllocDir: dir, Mode: artifactModeDir}
	err = p.getMinimal(context.Background())
	if err == nil || !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("unexpected error: %v", err)
	}
}
