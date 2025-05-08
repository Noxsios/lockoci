// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Contributors to lockoci

// Package lock provides an OCI locking mechanism
package lock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

// Media types for lockoci
const (
	StateFileMediaType  = "application/vnd.opentofu.state.v1+json"
	AnnotationLockState = "org.opentofu.state.locked"
)

// ErrLocked is a sentinel error when another client has locked the manifest for editing
var ErrLocked = errors.New("manifest is currently locked for editing")

// Acquire attempts to acquire a lock on a ref
func Acquire(ctx context.Context, repo *remote.Repository, ref registry.Reference, force bool) (v1.Manifest, error) {
	var stateInitialized bool

	if ref.Reference == "" {
		return v1.Manifest{}, fmt.Errorf("reference is blank")
	}

	// ignoring the error here
	_ = repo.Tags(ctx, "", func(tags []string) error {
		if slices.Contains(tags, ref.Reference) {
			stateInitialized = true
		}
		return nil
	})

	if !stateInitialized {
		emptyConfig := v1.DescriptorEmptyJSON
		err := repo.Push(ctx, emptyConfig, bytes.NewReader([]byte(`{}`)))
		if err != nil {
			return v1.Manifest{}, err
		}

		manifest := v1.Manifest{
			Versioned: specs.Versioned{
				SchemaVersion: 2,
			},
			MediaType: v1.MediaTypeImageManifest,
			Config:    emptyConfig,
			Layers:    []v1.Descriptor{},
		}

		manifest = lockManifest(manifest)

		manifestBytes, err := json.Marshal(manifest)
		if err != nil {
			return v1.Manifest{}, err
		}
		manifestDescriptor := content.NewDescriptorFromBytes(v1.MediaTypeImageManifest, manifestBytes)
		fmt.Println("init locked manifest", manifestDescriptor.Digest)
		err = repo.PushReference(ctx, manifestDescriptor, bytes.NewReader(manifestBytes), repo.Reference.Reference)
		if err != nil {
			return v1.Manifest{}, err
		}

		// trigger storage of etag
		desc, err := repo.Resolve(ctx, repo.Reference.Reference)
		if err != nil {
			return v1.Manifest{}, err
		}
		if desc.Digest != manifestDescriptor.Digest {
			return v1.Manifest{}, fmt.Errorf("expected %s, got %s", manifestDescriptor.Digest, desc.Digest)
		}

		return manifest, nil
	}

	var manifest v1.Manifest

	currentManifestDesc, currentManifestReadCloser, err := repo.FetchReference(ctx, ref.Reference)
	if err != nil {
		return v1.Manifest{}, err
	}
	defer currentManifestReadCloser.Close()

	if currentManifestDesc.MediaType != v1.MediaTypeImageManifest {
		return v1.Manifest{}, fmt.Errorf("unusable media type: expected %s, got %s", v1.MediaTypeImageManifest, currentManifestDesc.MediaType)
	}

	b, err := io.ReadAll(currentManifestReadCloser)
	if err != nil {
		return v1.Manifest{}, err
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		return v1.Manifest{}, err
	}

	if isLocked(manifest) && !force {
		return v1.Manifest{}, ErrLocked
	}

	manifest = lockManifest(manifest)

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return v1.Manifest{}, err
	}

	manifestDescriptor := content.NewDescriptorFromBytes(v1.MediaTypeImageManifest, manifestBytes)
	fmt.Println("locking manifest", manifestDescriptor.Digest)
	err = repo.PushReference(ctx, manifestDescriptor, bytes.NewReader(manifestBytes), ref.Reference)
	if err != nil {
		return v1.Manifest{}, err
	}

	return manifest, nil
}

// PushState pushes the contents of the reader as a new entry to the ref
func PushState(ctx context.Context, repo *remote.Repository, ref registry.Reference, reader io.Reader, force bool) error {
	b, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	expected := content.NewDescriptorFromBytes(StateFileMediaType, b)
	err = repo.Push(ctx, expected, bytes.NewReader(b))
	if err != nil {
		return err
	}

	manifest, err := Acquire(ctx, repo, ref, force)
	if err != nil {
		return err
	}

	manifest = unlockManifest(manifest)
	manifest.Layers = append(manifest.Layers, expected)

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	manifestDescriptor := content.NewDescriptorFromBytes(v1.MediaTypeImageManifest, manifestBytes)
	fmt.Println("pushing state", expected.Digest, "manifest", manifestDescriptor.Digest)
	err = repo.PushReference(ctx, manifestDescriptor, bytes.NewReader(manifestBytes), ref.Reference)
	if err != nil {
		return err
	}

	return nil
}

func lockManifest(current v1.Manifest) v1.Manifest {
	if current.Annotations == nil {
		current.Annotations = make(map[string]string)
	}
	current.Annotations[AnnotationLockState] = "true"
	return current
}

func unlockManifest(current v1.Manifest) v1.Manifest {
	if current.Annotations == nil {
		current.Annotations = make(map[string]string)
	}
	current.Annotations[AnnotationLockState] = "false"
	return current
}

func isLocked(current v1.Manifest) bool {
	return current.Annotations != nil && current.Annotations[AnnotationLockState] == "true"
}
