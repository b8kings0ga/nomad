// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package command

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func getJobFile(source string) (io.Reader, func(), error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing jobfile %q: %v", source, err)
	}
	if u.Scheme == "" || u.Scheme == "file" {
		path := source
		if u.Scheme == "file" {
			path = u.Path
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, nil, fmt.Errorf("Error opening file %q: %v", source, err)
		}
		return file, func() { file.Close() }, nil
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, nil, fmt.Errorf("nomad_min: unsupported feature jobfile scheme %q", u.Scheme)
	}
	resp, err := http.Get(source)
	if err != nil {
		return nil, nil, fmt.Errorf("Error getting jobfile from %q: %v", source, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("Error getting jobfile from %q: HTTP %s", source, resp.Status)
	}
	return resp.Body, func() { resp.Body.Close() }, nil
}
