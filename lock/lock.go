package lock

import (
	"bytes"
	"context"
	"os"

	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"
)

func Lock(ctx context.Context, ref string, path string) error {
	// This function is a placeholder for the lockoci functionality.
	// The actual implementation will depend on the specific requirements
	// of the lockoci tool.
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return err
	}
	repo.PlainHTTP = true
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	expected := content.NewDescriptorFromBytes("application/vnd.opentofu.lock.v1+json", b)
	err = repo.Push(ctx, expected, bytes.NewReader(b))
	if err != nil {
		return err
	}
	return nil
}
