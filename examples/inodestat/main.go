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

	dirf, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalln("error open mount:", err)
	}
	defer dirf.Close()

	u, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		log.Fatalln("error parsing inode:", err)
	}

	name, err := scoutfs.InoToPath(dirf, u)
	if err != nil {
		log.Fatalln("error getting pathname:", err)
	}

	f, err := scoutfs.OpenByID(dirf, u, os.O_RDONLY, name)
	if err != nil {
		log.Fatalln("error open by id:", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Fatalln("error stat:", err)
	}

	fmt.Println("Full name relative to mount:", name)
	fmt.Println("Name:", fi.Name())
	fmt.Println("Size:", fi.Size())
	fmt.Println("Mode:", fi.Mode())
}
