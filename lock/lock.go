// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Contributors to lockoci

// Package lock provides an OCI locking mechanism
package lock

import (
	"bytes"
	"context"
	"io"

	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

// Media types for lockoci
const (
	StateFileMediaType = "application/vnd.opentofu.state.v1+json"
	LockFileMediaType  = "application/vnd.opentofu.lock.v1+json"
)

// Lock uploads a file to an OCI registry, then locks it
func Lock(ctx context.Context, ref registry.Reference, reader io.Reader) error {
	repo, err := remote.NewRepository(ref.String())
	if err != nil {
		return err
	}
	repo.PlainHTTP = true

	b, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	expected := content.NewDescriptorFromBytes(StateFileMediaType, b)
	err = repo.Push(ctx, expected, bytes.NewReader(b))
	if err != nil {
		return err
	}
	return nil
}
