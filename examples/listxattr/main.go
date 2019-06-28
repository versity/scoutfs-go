// Copyright (c) 2018 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

package main

import (
	"fmt"
	"log"
	"os"

	scoutfs "github.com/versity/scoutfs-go"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "-h" {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "<filepath>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("open %v: %v", os.Args[1], err)
	}

	lxr := scoutfs.NewListXattrRaw(f)

	for {
		attrs, err := lxr.Next()
		if err != nil {
			log.Fatalf("next(): %v", err)
		}
		if attrs == nil {
			break
		}
		for _, attr := range attrs {
			fmt.Println(attr)
		}
	}
}
