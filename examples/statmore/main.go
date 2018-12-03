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

	s, err := scoutfs.StatMore(os.Args[1])
	if err != nil {
		log.Fatalln("error statmore:", err)
	}

	fmt.Printf("%+v\n", s)
}
