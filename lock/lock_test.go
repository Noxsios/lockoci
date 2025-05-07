// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Contributors to lockoci

package lock

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory" // used for docker test registry
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
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
		err := reg.ListenAndServe()
		require.NoError(t, err)
	}()
	return fmt.Sprintf("localhost:%d", port)
}

func TestLock(t *testing.T) {
	content := "Hello World!"

	registryURL := SetupInMemoryRegistry(t, 5005)

	r, err := remote.NewRegistry(registryURL)
	require.NoError(t, err)
	r.PlainHTTP = true
	err = r.Ping(t.Context())
	require.NoError(t, err)

	ref := orasRegistry.Reference{
		Registry:   registryURL,
		Repository: "testrepo",
		Reference:  "latest",
	}
	err = Lock(t.Context(), ref, strings.NewReader(content))
	require.NoError(t, err)
}
