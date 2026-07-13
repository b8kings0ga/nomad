// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package command

import (
	"fmt"
	"io"
	"os"

	gg "github.com/hashicorp/go-getter"
)

func getJobFile(source string) (io.Reader, func(), error) {
	tmp, err := os.CreateTemp("", "jobfile")
	if err != nil {
		return nil, nil, err
	}
	name := tmp.Name()
	cleanup := func() { os.Remove(name) }
	if err := tmp.Close(); err != nil {
		cleanup()
		return nil, nil, err
	}
	pwd, err := os.Getwd()
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	client := &gg.Client{Src: source, Pwd: pwd, Dst: name, DisableSymlinks: true}
	if err := client.Get(); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("Error getting jobfile from %q: %v", source, err)
	}
	file, err := os.Open(name)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("Error opening file %q: %v", source, err)
	}
	return file, func() {
		file.Close()
		os.Remove(name)
	}, nil
}
