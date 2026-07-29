// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package getter

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/helper/subproc"
)

func init() {
	subproc.Do(SubCommand, runMinimalGetter)
}

func runMinimalGetter() int {
	l := log.New(&log.LoggerOptions{Output: os.Stderr, DisableTime: true, Level: log.Debug})
	p := new(parameters)
	if err := p.read(os.Stdin); err != nil {
		subproc.Print("failed to read configuration: %v", err)
		return subproc.ExitFailure
	}
	ctx, cancel := subproc.Context(p.deadline())
	defer cancel()
	subproc.SetExpiration(ctx)
	if !p.DisableFilesystemIsolation {
		if err := lockdown(l, p.AllocDir, p.TaskDir, p.FilesystemIsolationExtraPaths); err != nil {
			subproc.Print("failed to sandbox %s process: %v", SubCommand, err)
			return subproc.ExitFailure
		}
	}
	if err := p.getMinimal(ctx); err != nil {
		subproc.Print("failed to download artifact: %v", err)
		return subproc.ExitFailure
	}
	if p.Chown {
		if err := chownDestination(p.Destination, p.User); err != nil {
			subproc.Print("failed to chown artifact: %v", err)
			return subproc.ExitFailure
		}
	}
	subproc.Print("artifact download was a success")
	return subproc.ExitSuccess
}

func (p *parameters) getMinimal(ctx context.Context) error {
	u, err := url.Parse(p.Source)
	if err != nil {
		return err
	}
	if u.Scheme != "" && u.Scheme != "file" && u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("nomad_min: unsupported feature artifact scheme %q", u.Scheme)
	}

	tmp, err := os.CreateTemp(p.AllocDir, "nomad-min-artifact-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if u.Scheme == "http" || u.Scheme == "https" {
		err = p.downloadHTTP(ctx, tmp, u.String())
	} else {
		path := u.Path
		if u.Scheme == "" {
			path = p.Source
		}
		err = copyLocalFile(tmp, path, p.HTTPMaxBytes)
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}

	archive := strings.ToLower(u.Path)
	if p.Mode == artifactModeFile || u.Query().Get("archive") == "false" {
		return installRaw(tmpName, p.Destination)
	}
	switch {
	case strings.HasSuffix(archive, ".tar.gz"), strings.HasSuffix(archive, ".tgz"):
		return extractTarGzip(tmpName, p.Destination, p)
	case strings.HasSuffix(archive, ".zip"):
		return extractZip(tmpName, p.Destination, p)
	case strings.HasSuffix(archive, ".gz"):
		return extractGzip(tmpName, p.Destination, strings.TrimSuffix(filepath.Base(u.Path), ".gz"), p)
	default:
		if p.Mode == artifactModeDir {
			return fmt.Errorf("nomad_min: unsupported feature artifact format %q", filepath.Ext(u.Path))
		}
		return installRaw(tmpName, p.Destination)
	}
}

func (p *parameters) downloadHTTP(ctx context.Context, dst io.Writer, source string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	for key, values := range p.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	client := &http.Client{Timeout: p.HTTPReadTimeout}
	if p.Insecure {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP status %s", resp.Status)
	}
	return copyLimited(dst, resp.Body, p.HTTPMaxBytes)
}

func copyLocalFile(dst io.Writer, path string, limit int64) error {
	src, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer src.Close()
	return copyLimited(dst, src, limit)
}

func copyLimited(dst io.Writer, src io.Reader, limit int64) error {
	if limit <= 0 {
		_, err := io.Copy(dst, src)
		return err
	}
	n, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err == nil && n > limit {
		return fmt.Errorf("artifact exceeds %d byte limit", limit)
	}
	return err
}

func installRaw(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, src)
	return errors.Join(copyErr, dst.Close())
}

func extractTarGzip(source, destination string, p *parameters) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	count, total := 0, int64(0)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		count++
		if err := checkArchiveLimits(p, count, total+h.Size); err != nil {
			return err
		}
		total += h.Size
		target, err := safeArchivePath(destination, h.Name)
		if err != nil {
			return err
		}
		switch h.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, 0755)
		case tar.TypeReg, tar.TypeRegA:
			err = writeArchiveFile(target, tr, h.FileInfo().Mode().Perm())
		default:
			return fmt.Errorf("nomad_min: unsupported feature archive entry type %d", h.Typeflag)
		}
		if err != nil {
			return err
		}
	}
}

func extractZip(source, destination string, p *parameters) error {
	r, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer r.Close()
	total := int64(0)
	for i, f := range r.File {
		total += int64(f.UncompressedSize64)
		if err := checkArchiveLimits(p, i+1, total); err != nil {
			return err
		}
		target, err := safeArchivePath(destination, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("nomad_min: unsupported feature archive symlink")
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = writeArchiveFile(target, rc, f.Mode().Perm())
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractGzip(source, destination, name string, p *parameters) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	if name == "" {
		name = "artifact"
	}
	target := destination
	if p.Mode == artifactModeDir || strings.HasSuffix(destination, string(os.PathSeparator)) {
		target = filepath.Join(destination, name)
	}
	var reader io.Reader = gz
	if p.DecompressionLimitSize > 0 {
		reader = io.LimitReader(gz, p.DecompressionLimitSize+1)
	}
	return writeArchiveFile(target, reader, 0644)
}

func safeArchivePath(root, name string) (string, error) {
	clean := filepath.Clean(name)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("artifact archive path escapes destination: %q", name)
	}
	return filepath.Join(root, clean), nil
}

func writeArchiveFile(path string, src io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	mode &^= os.ModeSetuid | os.ModeSetgid
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, src)
	return errors.Join(copyErr, dst.Close())
}

func checkArchiveLimits(p *parameters, count int, size int64) error {
	if p.DecompressionLimitFileCount > 0 && count > p.DecompressionLimitFileCount {
		return fmt.Errorf("artifact exceeds %d file limit", p.DecompressionLimitFileCount)
	}
	if p.DecompressionLimitSize > 0 && size > p.DecompressionLimitSize {
		return fmt.Errorf("artifact exceeds %d byte decompression limit", p.DecompressionLimitSize)
	}
	return nil
}
