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
	"strconv"

	scoutfs "github.com/versity/scoutfs-go"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] == "-h" {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "<scoutfs mount point> <inode>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalln("error open mount:", err)
	}
	defer f.Close()

	u, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		log.Fatalln("error parsing inode:", err)
	}

	s, err := scoutfs.GetParents(f, u, nil)
	if err != nil {
		log.Fatalln("error get parents:", err)
	}

	fmt.Println(s)
}
