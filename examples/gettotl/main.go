// Copyright (c) 2021 Versity Software, Inc.
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
	if len(os.Args) != 5 || os.Args[1] == "-h" {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "<scoutfs mount point> <id1> <id2> <id3>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalln("error open mount:", err)
	}
	defer f.Close()

	u1, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		log.Fatalln("error parsing id1:", err)
	}
	u2, err := strconv.ParseUint(os.Args[3], 10, 64)
	if err != nil {
		log.Fatalln("error parsing id2:", err)
	}
	u3, err := strconv.ParseUint(os.Args[4], 10, 64)
	if err != nil {
		log.Fatalln("error parsing id3:", err)
	}

	t, err := scoutfs.ReadXattrTotals(f, u1, u2, u3)
	if err != nil {
		log.Fatalln("error read totals:", err)
	}

	fmt.Println("xattrs match: ", t.Count)
	fmt.Println("total value : ", t.Total)
}
