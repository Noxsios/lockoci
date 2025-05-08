// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Contributors to lockoci

package lock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory" // used for docker test registry
	"github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/content"
	orasRegistry "oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

// SetupInMemoryRegistry sets up an in-memory registry on localhost and returns the address.
func SetupInMemoryRegistry(t *testing.T, port int) string {
	t.Helper()
	config := &configuration.Configuration{}
	config.HTTP.Addr = fmt.Sprintf(":%d", port)
	config.Log.AccessLog.Disabled = true
	config.Log.Level = "error"
	logrus.SetOutput(io.Discard)
	config.HTTP.DrainTimeout = 10 * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]any{}}
	reg, err := registry.NewRegistry(t.Context(), config)
	require.NoError(t, err)
	go func() {
		_ = reg.ListenAndServe()
	}()
	return fmt.Sprintf("localhost:%d", port)
}

func TestLock(t *testing.T) {
	registryURL := SetupInMemoryRegistry(t, 5007)

	ref := orasRegistry.Reference{
		Registry:   registryURL,
		Repository: "testrepo",
		Reference:  "latest",
	}

	repo, err := remote.NewRepository(ref.String())
	require.NoError(t, err)
	repo.PlainHTTP = true
	repo.Client = &etagClient{}

	err = PushState(t.Context(), repo, strings.NewReader("Hello World!"), false)
	require.NoError(t, err)

	// acquire a lock but never unlock
	_, err = Acquire(t.Context(), repo, false)
	require.NoError(t, err)

	err = PushState(t.Context(), repo, strings.NewReader("Why Hello There!"), false)
	require.EqualError(t, err, ErrLocked.Error())

	// force overwrite
	err = PushState(t.Context(), repo, strings.NewReader("Why Hello There!"), true)
	require.NoError(t, err)

	// trigger a push collision
	// err = triggerPushCollision(t.Context(), repo)
	// require.ErrorContains(t, err, "412")
}

// TODO: remove me if a workaround is figured out
func triggerPushCollision(ctx context.Context, repo *remote.Repository) error {
	emptyConfig := v1.DescriptorEmptyJSON

	err := repo.Push(ctx, emptyConfig, bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return err
	}

	m1 := v1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: v1.MediaTypeImageManifest,
		Config:    emptyConfig,
		Layers:    []v1.Descriptor{},
		Annotations: map[string]string{
			"foo": "bar", // changeup sha
		},
	}

	manifestBytes, err := json.Marshal(m1)
	if err != nil {
		return err
	}

	m1Descriptor := content.NewDescriptorFromBytes(v1.MediaTypeImageManifest, manifestBytes)
	err = repo.PushReference(ctx, m1Descriptor, bytes.NewReader(manifestBytes), repo.Reference.Reference)
	if err != nil {
		return err
	}

	// trigger storage of etag
	_, err = repo.Resolve(ctx, repo.Reference.Reference)
	if err != nil {
		return err
	}

	m2 := v1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: v1.MediaTypeImageManifest,
		Config:    emptyConfig,
		Layers:    []v1.Descriptor{},
		Annotations: map[string]string{
			"bar": "baz", // changeup sha
		},
	}

	manifestBytes, err = json.Marshal(m2)
	if err != nil {
		return err
	}

	m2Descriptor := content.NewDescriptorFromBytes(v1.MediaTypeImageManifest, manifestBytes)

	// configure an incorrect etag
	// this simulates another client already pushed to the tag
	repo.Client.(*etagClient).etags.Store("latest", fmt.Sprintf(`"%s"`, m2Descriptor.Digest.String()))

	// this should trigger an error but it won't
	return repo.PushReference(ctx, m2Descriptor, bytes.NewReader(manifestBytes), repo.Reference.Reference)
}
