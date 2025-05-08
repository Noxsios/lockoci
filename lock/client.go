// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Contributors to lockoci

package lock

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

type etagClient struct {
	Client remote.Client

	etags sync.Map
}

func (c *etagClient) client() remote.Client {
	if c.Client == nil {
		return auth.DefaultClient
	}
	return c.Client
}

func (c *etagClient) Do(originalReq *http.Request) (*http.Response, error) {
	ctx := originalReq.Context()
	req := originalReq.Clone(ctx)

	fmt.Println(">", req.Method, req.URL)

	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	n := len(parts)
	var lastTwo [2]string
	if n >= 2 {
		lastTwo[0] = parts[n-2]
		lastTwo[1] = parts[n-1]
	}

	// get the last two path elements of the request
	if req.Method == http.MethodPut {
		if lastTwo[0] == "manifests" {
			got, ok := c.etags.Load(lastTwo[1])
			if ok {
				etag := got.(string) // bad type casting

				fmt.Printf("> setting If-None-Match for %s to %s\n", lastTwo[1], etag)
				// https://github.com/distribution/distribution/blob/95647cba1d992a4da614bbae725022b5215c98c4/internal/client/repository.go#L506
				req.Header.Add(http.CanonicalHeaderKey("If-None-Match"), etag)
				// ^ but this is actually not respected anywhere in https://github.com/distribution/distribution/blob/95647cba1d992a4da614bbae725022b5215c98c4/registry/handlers/manifests.go#L238
				//
				// etag is only used in https://github.com/distribution/distribution/blob/95647cba1d992a4da614bbae725022b5215c98c4/registry/handlers/manifests.go#L132
				// for returning a 312 on GETs
			}
		}
	}

	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}

	fmt.Println("<", resp.Status)

	etag := resp.Header.Get("ETag")
	if etag != "" {
		fmt.Println("<", "ETag:", resp.Header.Get("ETag"))
		if lastTwo[0] == "manifests" {
			fmt.Printf("< storing %s as %s\n", lastTwo[1], etag)
			c.etags.Store(lastTwo[1], etag)
		}
	}

	fmt.Println()

	return resp, err
}
