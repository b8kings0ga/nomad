package nomad

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

func TestRatatoskrPackageAdmissionUsesRaftVariable(t *testing.T) {
	srv, cleanup := TestServer(t, nil)
	defer cleanup()
	testutil.WaitForLeader(t, srv.RPC)
	testutil.WaitForKeyring(t, srv.RPC, srv.config.Region)

	policyNode := &Node{srv: srv}
	snapshot, err := srv.fsm.State().Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	approved := strings.Repeat("ab", 32)
	rejected := strings.Repeat("cd", 32)
	if err := policyNode.checkRatatoskrPackageAdmission(snapshot, &structs.Node{}); err != nil {
		t.Fatalf("missing policy changed normal registration: %v", err)
	}

	items, err := json.Marshal(map[string]string{"version": "1", "allowed_sha256": approved})
	if err != nil {
		t.Fatal(err)
	}
	encrypted, keyID, err := srv.encrypter.Encrypt(items)
	if err != nil {
		t.Fatal(err)
	}
	response := srv.fsm.State().VarSet(structs.VarApplyStateRequestType, 100, &structs.VarApplyStateRequest{
		Op: structs.VarOpSet,
		Var: &structs.VariableEncrypted{
			VariableMetadata: structs.VariableMetadata{Namespace: ratatoskrAdmissionVariableNamespace, Path: ratatoskrAdmissionVariablePath},
			VariableData:     structs.VariableData{Data: encrypted, KeyID: keyID},
		},
	})
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	snapshot, err = srv.fsm.State().Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		sha           string
		clientVersion string
		want          bool
	}{
		{name: "approved", sha: approved, clientVersion: "1", want: true},
		{name: "old package", sha: rejected, clientVersion: "1"},
		{name: "missing package", clientVersion: "1"},
		{name: "old Nomad client", sha: approved},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := policyNode.checkRatatoskrPackageAdmission(snapshot, &structs.Node{
				Meta:       map[string]string{ratatoskrAdmissionPackageMetaKey: tc.sha},
				Attributes: map[string]string{ratatoskrAdmissionClientAttribute: tc.clientVersion},
			})
			if (err == nil) != tc.want {
				t.Fatalf("admission error = %v, want allowed=%t", err, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		sha  string
		want bool
	}{
		{sha: rejected},
		{sha: approved, want: true},
	} {
		node := mock.Node()
		node.Meta[ratatoskrAdmissionPackageMetaKey] = tc.sha
		node.Attributes[ratatoskrAdmissionClientAttribute] = "1"
		request := &structs.NodeRegisterRequest{Node: node, WriteRequest: structs.WriteRequest{Region: srv.config.Region}}
		var reply structs.NodeUpdateResponse
		err := NewNodeEndpoint(srv, nil).Register(request, &reply)
		if (err == nil) != tc.want {
			t.Fatalf("Node.Register sha=%q error=%v, want allowed=%t", tc.sha, err, tc.want)
		}
		if tc.want {
			rolledBack := node.Copy()
			rolledBack.Meta[ratatoskrAdmissionPackageMetaKey] = rejected
			request.Node = rolledBack
			if err := NewNodeEndpoint(srv, nil).Register(request, &reply); err == nil {
				t.Fatal("previously registered node rejoined with a disallowed package")
			}
		}
	}
	updatedItems, err := json.Marshal(map[string]string{"version": "1", "allowed_sha256": rejected})
	if err != nil {
		t.Fatal(err)
	}
	updated, updatedKeyID, err := srv.encrypter.Encrypt(updatedItems)
	if err != nil {
		t.Fatal(err)
	}
	response = srv.fsm.State().VarSet(structs.VarApplyStateRequestType, 101, &structs.VarApplyStateRequest{
		Op: structs.VarOpSet,
		Var: &structs.VariableEncrypted{
			VariableMetadata: structs.VariableMetadata{Namespace: ratatoskrAdmissionVariableNamespace, Path: ratatoskrAdmissionVariablePath},
			VariableData:     structs.VariableData{Data: updated, KeyID: updatedKeyID},
		},
	})
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	snapshot, err = srv.fsm.State().Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := policyNode.checkRatatoskrPackageAdmission(snapshot, &structs.Node{Meta: map[string]string{ratatoskrAdmissionPackageMetaKey: rejected}, Attributes: map[string]string{ratatoskrAdmissionClientAttribute: "1"}}); err != nil {
		t.Fatalf("policy update did not admit new package: %v", err)
	}
	if err := policyNode.checkRatatoskrPackageAdmission(snapshot, &structs.Node{Meta: map[string]string{ratatoskrAdmissionPackageMetaKey: approved}, Attributes: map[string]string{ratatoskrAdmissionClientAttribute: "1"}}); err == nil {
		t.Fatal("policy update still admitted previous package")
	}
}

func TestParseRatatoskrAdmissionPolicyRejectsInvalidConfig(t *testing.T) {
	for _, raw := range []string{
		`not json`,
		`{"version":"2","allowed_sha256":"` + strings.Repeat("ab", 32) + `"}`,
		`{"version":"1","allowed_sha256":""}`,
		`{"version":"1","allowed_sha256":"bad"}`,
	} {
		if _, err := parseRatatoskrAdmissionPolicy([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid policy: %s", raw)
		}
	}
}
