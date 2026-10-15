// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package structs

import (
	"strings"
	"testing"
)

func TestMimirZeroDisk(t *testing.T) {
	disk := &EphemeralDisk{SizeMB: 0}
	if err := disk.Validate(); err != nil {
		t.Fatal(err)
	}
	logs := DefaultLogConfig()
	logs.Disabled = true
	if err := logs.Validate(disk); err != nil {
		t.Fatal(err)
	}
	logs.Disabled = false
	if err := logs.Validate(disk); err == nil {
		t.Fatal("local logs accepted without disk")
	}
	disk.Sticky = true
	if err := disk.Validate(); err == nil {
		t.Fatal("sticky zero disk accepted")
	}
	disk.Sticky = false
	disk.Migrate = true
	if err := disk.Validate(); err == nil {
		t.Fatal("migrating zero disk accepted")
	}
	disk.Migrate = false
	disk.SizeMB = -1
	if err := disk.Validate(); err == nil {
		t.Fatal("negative disk accepted")
	}
}

func TestMimirZeroDiskRestrictedToStatelessSandbox(t *testing.T) {
	for _, tc := range []struct {
		driver, mode    string
		disabled, allow bool
	}{
		{"mimc", "sandbox", true, true}, {"mimc", "sandbox", false, false}, {"mimc", "raw", true, false}, {"raw_exec", "sandbox", true, false},
	} {
		tg := &TaskGroup{Name: "service", Count: 1, EphemeralDisk: &EphemeralDisk{SizeMB: 0}, Tasks: []*Task{{Name: "service", Driver: tc.driver, Config: map[string]interface{}{"mode": tc.mode}, LogConfig: &LogConfig{Disabled: tc.disabled, MaxFiles: 10, MaxFileSizeMB: 10}}}}
		job := &Job{Type: JobTypeService, TaskGroups: []*TaskGroup{tg}}
		job.Canonicalize()
		err := tg.Validate(job)
		if tc.allow && err != nil {
			t.Fatalf("canonical stateless sandbox was rejected: %v", err)
		}
		rejected := err != nil && strings.Contains(err.Error(), "zero ephemeral disk requires")
		if rejected == tc.allow {
			t.Fatalf("driver=%s mode=%s disabled=%v err=%v", tc.driver, tc.mode, tc.disabled, err)
		}
	}
}
