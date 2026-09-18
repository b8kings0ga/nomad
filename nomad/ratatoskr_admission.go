package nomad

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
)

const ratatoskrAdmissionVariableNamespace = "default"
const ratatoskrAdmissionVariablePath = "mimir/ratatoskr-admission"
const ratatoskrAdmissionPackageMetaKey = "mim_ratatoskr_package_sha256"
const ratatoskrAdmissionClientAttribute = "mim.ratatoskr_admission_version"

type ratatoskrAdmissionPolicy struct {
	allowed map[string]struct{}
}

func parseRatatoskrAdmissionPolicy(data []byte) (ratatoskrAdmissionPolicy, error) {
	var items map[string]string
	if err := json.Unmarshal(data, &items); err != nil || items["version"] != "1" {
		return ratatoskrAdmissionPolicy{}, errors.New("invalid Ratatoskr admission variable")
	}
	policy := ratatoskrAdmissionPolicy{allowed: make(map[string]struct{})}
	for _, raw := range strings.Split(items["allowed_sha256"], ",") {
		sha := strings.ToLower(strings.TrimSpace(raw))
		decoded, err := hex.DecodeString(sha)
		if err != nil || len(decoded) != 32 {
			return ratatoskrAdmissionPolicy{}, errors.New("invalid Ratatoskr admission package SHA")
		}
		policy.allowed[sha] = struct{}{}
	}
	if len(policy.allowed) == 0 {
		return ratatoskrAdmissionPolicy{}, errors.New("empty Ratatoskr admission package set")
	}
	return policy, nil
}

// The absent variable preserves Nomad's original registration behavior. Once
// installed, the Raft-backed variable applies equally to old and new node IDs.
func (n *Node) checkRatatoskrPackageAdmission(snap *state.StateSnapshot, node *structs.Node) error {
	entry, err := snap.GetVariable(nil, ratatoskrAdmissionVariableNamespace, ratatoskrAdmissionVariablePath)
	if err != nil || entry == nil {
		return err
	}
	plaintext, err := n.srv.encrypter.Decrypt(entry.Data, entry.KeyID)
	if err != nil {
		return fmt.Errorf("decrypt Ratatoskr admission variable: %w", err)
	}
	policy, err := parseRatatoskrAdmissionPolicy(plaintext)
	if err != nil {
		return err
	}
	if node.Attributes[ratatoskrAdmissionClientAttribute] != "1" {
		return fmt.Errorf("Nomad client lacks Ratatoskr admission support: %w", structs.ErrPermissionDenied)
	}
	sha := strings.ToLower(strings.TrimSpace(node.Meta[ratatoskrAdmissionPackageMetaKey]))
	if _, allowed := policy.allowed[sha]; !allowed {
		return fmt.Errorf("Ratatoskr package %q is not admitted: %w", sha, structs.ErrPermissionDenied)
	}
	return nil
}
