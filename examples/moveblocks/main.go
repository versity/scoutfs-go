// Copyright (c) 2021 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

package main

import (
	"fmt"
	"os"

	scoutfs "github.com/versity/scoutfs-go"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] == "-h" {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "<file_to_move_blocks_from> <file_to_append_blocks_to>")
		os.Exit(1)
	}

	ffrom, err := os.OpenFile(os.Args[1], os.O_RDWR, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %v: %v\n", os.Args[1], err)
		os.Exit(1)
	}
	defer ffrom.Close()

	fto, err := os.OpenFile(os.Args[2], os.O_RDWR, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %v: %v\n", os.Args[2], err)
		os.Exit(1)
	}
	defer fto.Close()

	err = scoutfs.MoveData(ffrom, fto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "move blocks: %v\n", err)
		os.Exit(1)
	}
}
