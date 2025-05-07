// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Harry Randazzo

// Package main is the entrypoint for the lockoci CLI
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"slices"
)

func main() {
	var help bool
	flag.BoolVar(&help, "h", false, "Print this message and exit.")

	var ver bool
	flag.BoolVar(&ver, "v", false, "Print the version number of sizeoci and exit.")

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
	
	fmt.Println("lockoci")
}