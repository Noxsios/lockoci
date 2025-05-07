// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

package lock

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory" // used for docker test registry
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}
	reg, err := registry.NewRegistry(t.Context(), config)
	require.NoError(t, err)
	//nolint:errcheck // ignore
	go reg.ListenAndServe()
	return fmt.Sprintf("localhost:%d", port)
}

func TestLock(t *testing.T) {
	content := "hello world"
	filename := "helloworld.txt"
	t.Cleanup(func() {
		_ = os.Remove(filename)
	})
	err := os.WriteFile(filename, []byte(content), 0600)
	assert.NoError(t, err)
	reg := SetupInMemoryRegistry(t, 5005)
	err = Lock(t.Context(), fmt.Sprintf("%s/%s", reg, "testrepo"), filename)
	assert.NoError(t, err)
}
