package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/versity/scoutfs-go"
)

func main() {
	// flags mount path name
	mountPath := flag.String("mount", "", "mount path name")
	// flags index type uint64
	indexType := flag.Uint("type", 0, "index type")
	// flags index start uint64
	indexStart := flag.Uint64("start", 0, "index start")
	// flags index end uint64
	indexEnd := flag.Uint64("end", 0, "index end")
	// flags show filenames
	resolveNames := flag.Bool("resolve", false, "show filenames")

	flag.Parse()

	f, err := os.Open(*mountPath)
	if err != nil {
		log.Fatalf("Error opening mount: %s", err)
	}
	defer f.Close()

	if *indexType > 255 {
		log.Fatal("index type out of bounds")
	}
	itype := uint8(*indexType)

	idx := scoutfs.NewIndexSearch(f, itype, *indexStart, *indexEnd)
	for {
		ents, err := idx.Next()
		if err != nil {
			log.Fatalf("Error reading index: %v", err)
		}
		if ents == nil {
			break
		}

		for _, e := range ents {
			if *resolveNames {
				name, err := scoutfs.InoToPath(f, e.Inode)
				if err != nil {
					log.Fatalf("Error resolving name for %v: %V", e.Inode, err)
					continue
				}
				fmt.Println(name)
				continue
			}
			fmt.Println(e.Inode, "=", e.Value)
		}
	}
}
