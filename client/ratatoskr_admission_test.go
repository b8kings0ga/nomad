// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/nomad/nomad/structs"
)

func TestRatatoskrPackageForRegistrationReplacesStaleMetadata(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "package.sha256")
	sha := strings.Repeat("ab", 32)
	if err := os.WriteFile(marker, []byte(strings.ToUpper(sha)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	node := &structs.Node{Meta: map[string]string{ratatoskrPackageMetaKey: strings.Repeat("cd", 32), "other": "retained"}}
	setRatatoskrPackageForRegistration(node, marker)
	if got := node.Meta[ratatoskrPackageMetaKey]; got != sha {
		t.Fatalf("package SHA = %q, want %q", got, sha)
	}
	if node.Meta["other"] != "retained" {
		t.Fatal("unrelated node metadata was changed")
	}
	if node.Attributes[ratatoskrAdmissionClientAttribute] != "1" {
		t.Fatal("registration did not declare admission-capable Nomad client")
	}
	if err := os.WriteFile(marker, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	setRatatoskrPackageForRegistration(node, marker)
	if _, ok := node.Meta[ratatoskrPackageMetaKey]; ok {
		t.Fatal("stale package SHA survived an invalid installed marker")
	}
}
