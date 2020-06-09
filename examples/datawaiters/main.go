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

	"github.com/versity/scoutfs-go"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "-h" {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "<scoutfs mount point>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("Open %v: %v", os.Args[1], err)
	}
	defer f.Close()

	w := scoutfs.NewWaiters(f)

	for {
		ents, err := w.Next()
		if err != nil {
			log.Fatalf("next(): %v", err)
		}
		if ents == nil {
			break
		}
		for _, ent := range ents {
			log.Printf("%+v", ent)
		}
	}
}
