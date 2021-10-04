// Copyright (c) 2021 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

// This example returns the values of a "group" of totl counts/values
// given a 2 uint64 id group tuple.  Or it will return the single value
// for a unique 3 uint64 id triple.

// id1    id2    id3
// |--group--|
// |--specific id---|

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	scoutfs "github.com/versity/scoutfs-go"
)

func main() {
	if (len(os.Args) != 4 && len(os.Args) != 5) || os.Args[1] == "-h" {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0],
			"<scoutfs mount point> <id1> <id2> or <scoutfs mount point> <id1> <id2> <id3>")
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

	if len(os.Args) == 4 {
		tg := scoutfs.NewTotalsGroup(f, u1, u2, 10)
		for {
			ttls, err := tg.Next()
			if err != nil {
				log.Fatalln("error read totals:", err)
			}
			if ttls == nil {
				break
			}
			for _, t := range ttls {
				fmt.Println("id:", t.ID[2], "xattrs match:", t.Count, "total value:", t.Total)
			}
		}
		return
	}

	u3, err := strconv.ParseUint(os.Args[4], 10, 64)
	if err != nil {
		log.Fatalln("error parsing id2:", err)
	}

	t, err := scoutfs.ReadXattrTotals(f, u1, u2, u3)
	if err != nil {
		log.Fatalln("error reading totals:", err)
	}
	fmt.Println("Count", t.Count)
	fmt.Println("Total", t.Total)
}
