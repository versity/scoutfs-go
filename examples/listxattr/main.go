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

// # can list more attributes than getfattr
// touch /mnt/scoutfs/file
// for i in $(seq 1 10000); do  setfattr -n "user.lots-$i" /mnt/scoutfs/file; done
// ./listxattr /mnt/scoutfs/file | wc -l
//
// # can list scoutfs hidden attrs
// touch /mnt/scoutfs/file
// setfattr -n scoutfs.hide.invisible /mnt/scoutfs/file
// ./listxattr /mnt/scoutfs/file

func main() {
	if len(os.Args) != 2 || os.Args[1] == "-h" {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "<filepath>")
		os.Exit(1)
	}

	f, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0)
	if err != nil {
		log.Fatalf("open %v: %v", os.Args[1], err)
	}

	b := make([]byte, 256*1024)

	lxr := scoutfs.NewListXattrHidden(f, b)

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
