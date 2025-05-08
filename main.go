// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Contributors to lockoci

// Package main is the entrypoint for the lockoci CLI
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"slices"
	"syscall"

	"github.com/noxsios/lockoci/lock"
	"oras.land/oras-go/v2/registry/remote"
)

func main() {
	var help bool
	flag.BoolVar(&help, "h", false, "Print this message and exit.")

	var ver bool
	flag.BoolVar(&ver, "v", false, "Print the version number of lockoci and exit.")

	var plainHTTP bool
	flag.BoolVar(&plainHTTP, "plain-http", false, "Allow insecure connections to registry without SSL check")

	var force bool
	flag.BoolVar(&force, "force", false, "Overwrite statefile lock and push new state")

	flag.Parse()

	if help || slices.Contains(os.Args[1:], "--help") {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if ver {
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Println("version information not available")
			os.Exit(1)
		}
		fmt.Println(bi.Main.Version)
		os.Exit(0)
	}

	if len(flag.Args()) > 2 || len(flag.Args()) == 0 {
		fatal(fmt.Errorf("invalid number of args: want 2, got %d", len(flag.Args())))
	}

	repo, err := remote.NewRepository(flag.Args()[0])
	if err != nil {
		fatal(err)
	}
	repo.PlainHTTP = plainHTTP

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	f, err := os.Open(flag.Args()[1])
	if err != nil {
		fatal(err)
	}

	if err := lock.PushState(ctx, repo, f, force); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
