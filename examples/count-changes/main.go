// Copyright (c) 2018 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	scoutfs "github.com/versity/scoutfs-go"
)

const (
	max32 = 0xffffffff
	max64 = 0xffffffffffffffff
)

type server struct {
	update    <-chan int
	lastcount int
}

func queryPopulation(basedir string, update chan<- int) error {
	f, err := os.Open(basedir)
	if err != nil {
		return err
	}
	defer f.Close()

	min := scoutfs.InodesEntry{}
	max := scoutfs.InodesEntry{Major: max64, Minor: max32, Ino: max64}
	h := scoutfs.NewQuery(f, scoutfs.ByMSeq(min, max))

	count := 0
	for {
		for {
			qents, err := h.Next()
			if err != nil {
				return fmt.Errorf("scoutfs next: %v", err)
			}
			if len(qents) == 0 {
				break
			}
			for _, qent := range qents {
				fmt.Printf("%#v\n", qent)
				count++
			}
		}

		update <- count
		time.Sleep(500 * time.Millisecond)
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	select {
	case x, ok := <-s.update:
		if ok {
			s.lastcount = x
		} else {
			fmt.Println("Channel closed!")
			os.Exit(1)
		}
	default:
	}

	fmt.Fprintf(w, "%s %d", `
		<HEAD><meta HTTP-EQUIV="REFRESH" content="1"></HEAD>
		<h1>SCOUTFS</h1></p>
		Files/Directories updated:`, s.lastcount)
}

func main() {
	if len(os.Args) != 2 || os.Args[1] == "-h" {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "<scoutfs mount point>")
		os.Exit(1)
	}

	// create server
	update := make(chan int, 1)
	s := &server{update: update}

	// run puplation query in separate goroutine
	go queryPopulation(os.Args[1], update)

	// run webserver
	log.Fatal(http.ListenAndServe(":8080", s))
}
