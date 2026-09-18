package client

import (
	"encoding/hex"
	"os"
	"strings"

	"github.com/hashicorp/nomad/nomad/structs"
)

const ratatoskrPackageMarkerPath = "/usr/local/bin/.mim-ratatoskr-package.sha256"
const ratatoskrPackageMetaKey = "mim_ratatoskr_package_sha256"
const ratatoskrAdmissionClientAttribute = "mim.ratatoskr_admission_version"

// Read the installed package marker at registration time. Dynamic node
// metadata can outlive a package downgrade, so it is not an admission proof.
func setRatatoskrPackageForRegistration(node *structs.Node, markerPath string) {
	if node == nil {
		return
	}
	if node.Meta == nil {
		node.Meta = make(map[string]string)
	}
	if node.Attributes == nil {
		node.Attributes = make(map[string]string)
	}
	node.Attributes[ratatoskrAdmissionClientAttribute] = "1"
	delete(node.Meta, ratatoskrPackageMetaKey)
	body, err := os.ReadFile(markerPath)
	if err != nil {
		return
	}
	sha := strings.ToLower(strings.TrimSpace(string(body)))
	decoded, err := hex.DecodeString(sha)
	if err != nil || len(decoded) != 32 {
		return
	}
	node.Meta[ratatoskrPackageMetaKey] = sha
}
