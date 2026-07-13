// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package nomad

import (
	"context"
	"strings"
	"testing"

	kms "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-kms-wrapping/v2/aead"
	"github.com/hashicorp/nomad/nomad/structs"
)

func TestMinimalAEADWrapperRoundTrip(t *testing.T) {
	e := new(Encrypter)
	w, err := e.newKMSWrapper(&structs.KEKProviderConfig{Provider: structs.KEKProviderAEAD}, "key-1", make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	blob, err := w.Encrypt(context.Background(), []byte("secret"), kms.WithAad([]byte("aad")))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := w.Decrypt(context.Background(), blob, kms.WithAad([]byte("aad")))
	if err != nil || string(plain) != "secret" {
		t.Fatalf("got %q, err %v", plain, err)
	}
}

func TestMinimalAEADWrapperRejectsCloud(t *testing.T) {
	e := new(Encrypter)
	_, err := e.newKMSWrapper(&structs.KEKProviderConfig{Provider: "awskms"}, "key-1", make([]byte, 32))
	if err == nil || !strings.Contains(err.Error(), "nomad_min: unsupported feature") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMinimalAEADWrapperUpstreamCompatibility(t *testing.T) {
	key := make([]byte, 32)
	e := new(Encrypter)
	minimal, err := e.newKMSWrapper(&structs.KEKProviderConfig{Provider: structs.KEKProviderAEAD}, "key-1", key)
	if err != nil {
		t.Fatal(err)
	}
	upstream := aead.NewWrapper()
	_, _ = upstream.SetConfig(context.Background(), kms.WithKeyId("key-1"))
	if err := upstream.SetAesGcmKeyBytes(key); err != nil {
		t.Fatal(err)
	}
	aad := kms.WithAad([]byte("raft-aad"))

	blob, err := upstream.Encrypt(context.Background(), []byte("from-full"), aad)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := minimal.Decrypt(context.Background(), blob, aad)
	if err != nil || string(plain) != "from-full" {
		t.Fatalf("full to minimal: got %q, err %v", plain, err)
	}

	blob, err = minimal.Encrypt(context.Background(), []byte("from-minimal"), aad)
	if err != nil {
		t.Fatal(err)
	}
	plain, err = upstream.Decrypt(context.Background(), blob, aad)
	if err != nil || string(plain) != "from-minimal" {
		t.Fatalf("minimal to full: got %q, err %v", plain, err)
	}
}
