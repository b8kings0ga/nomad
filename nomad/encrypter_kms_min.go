// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package nomad

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	kms "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/structs/config"
)

func fallbackVaultConfig(*structs.KEKProviderConfig, *config.VaultConfig) {}

func (e *Encrypter) newKMSWrapper(provider *structs.KEKProviderConfig, keyID string, kek []byte) (kms.Wrapper, error) {
	if provider.Provider != "" && provider.Provider != structs.KEKProviderAEAD {
		return nil, fmt.Errorf("nomad_min: unsupported feature keyring provider %q", provider.Provider)
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &minimalAEADWrapper{keyID: keyID, aead: gcm}, nil
}

// minimalAEADWrapper preserves the wire representation used by the upstream
// go-kms AEAD wrapper: Ciphertext is nonce || AES-GCM-sealed-data. BlobInfo is
// retained because it is part of the persisted keyring/Raft schema.
type minimalAEADWrapper struct {
	keyID string
	aead  cipher.AEAD
}

func (*minimalAEADWrapper) Type(context.Context) (kms.WrapperType, error) {
	return kms.WrapperTypeAead, nil
}

func (w *minimalAEADWrapper) KeyId(context.Context) (string, error) { return w.keyID, nil }

func (w *minimalAEADWrapper) SetConfig(_ context.Context, options ...kms.Option) (*kms.WrapperConfig, error) {
	opts, err := kms.GetOpts(options...)
	if err != nil {
		return nil, err
	}
	if opts.WithKeyId != "" {
		w.keyID = opts.WithKeyId
	}
	return &kms.WrapperConfig{Metadata: map[string]string{"aead_type": kms.AeadTypeAesGcm.String()}}, nil
}

func (w *minimalAEADWrapper) Encrypt(_ context.Context, plaintext []byte, options ...kms.Option) (*kms.BlobInfo, error) {
	if plaintext == nil {
		return nil, errors.New("given plaintext for encryption is nil")
	}
	opts, err := kms.GetOpts(options...)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, w.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	sealed := w.aead.Seal(nil, nonce, plaintext, opts.WithAad)
	return &kms.BlobInfo{
		Ciphertext: append(nonce, sealed...),
		KeyInfo:    &kms.KeyInfo{KeyId: w.keyID},
	}, nil
}

func (w *minimalAEADWrapper) Decrypt(_ context.Context, blob *kms.BlobInfo, options ...kms.Option) ([]byte, error) {
	if blob == nil {
		return nil, errors.New("given ciphertext for decryption is nil")
	}
	if len(blob.Ciphertext) < w.aead.NonceSize() {
		return nil, errors.New("ciphertext is shorter than AES-GCM nonce")
	}
	opts, err := kms.GetOpts(options...)
	if err != nil {
		return nil, err
	}
	nonceSize := w.aead.NonceSize()
	return w.aead.Open(nil, blob.Ciphertext[:nonceSize], blob.Ciphertext[nonceSize:], opts.WithAad)
}
