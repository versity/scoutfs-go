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
		log.Fatalf("open %q: %v", os.Args[1], err)
	}
	defer f.Close()

	id, err := scoutfs.GetIDs(f)
	if err != nil {
		log.Fatalf("error GetIDs: %v", err)
	}

	fmt.Printf("fsid %016x\n", id.FSID)
	fmt.Printf("rid  %016x\n", id.RandomID)
	fmt.Printf("%v\n", id.ShortID)
}
