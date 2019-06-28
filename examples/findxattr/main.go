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
	if len(os.Args) != 3 || os.Args[1] == "-h" {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "<mount point> <xattr key>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("open %v: %v", os.Args[1], err)
	}

	q := scoutfs.NewXattrQuery(f, os.Args[2])

	for {
		inodes, err := q.Next()
		if err != nil {
			log.Fatalf("Next(): %v", err)
		}
		if inodes == nil {
			break
		}
		for _, inode := range inodes {
			fmt.Println(inode)
		}
	}
}
