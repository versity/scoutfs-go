package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"time"
	"unsafe"

	scoutfs "github.com/versity/scoutfs-go"
)

type inoState struct {
	metaSeq uint64
}

func main() {
	path := flag.String("path", "", "path to scoutfs mount point")
	debug := flag.Bool("debug", false, "enable verbose debugging output")

	var xattrNames stringSlice
	flag.Var(&xattrNames, "xattr", "xattr name to read with inodes (can be repeated)")

	flag.Parse()

	if *path == "" {
		fmt.Fprintf(os.Stderr, "error: -path is required\n")
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Open(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening %s: %v\n", *path, err)
		os.Exit(1)
	}
	defer f.Close()

	// Track known inodes and their meta_seq
	known := make(map[uint64]*inoState)
	lines := 0

	// Allocate a reusable results buffer for RawReadInodeInfo
	resultsBuf := make([]byte, 1024*1024)

	for {
		reader := scoutfs.NewRawMetaSeqReader(f)

		var allMS []scoutfs.MetaSeqEntry
		for {
			items, err := reader.Next()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading meta_seq: %v\n", err)
				os.Exit(1)
			}
			if items == nil {
				break
			}
			allMS = append(allMS, items...)
		}

		// Find inodes that differ from our known state
		var diffInos []uint64
		seen := make(map[uint64]uint64) // ino -> meta_seq from current scan

		for _, ms := range allMS {
			seen[ms.Ino] = ms.Seq
		}

		// Check for new or changed inodes
		for ino, metaSeq := range seen {
			st, ok := known[ino]
			if !ok || st.metaSeq != metaSeq {
				diffInos = append(diffInos, ino)
			}
		}

		// Check for removed inodes
		for ino := range known {
			if _, ok := seen[ino]; !ok {
				diffInos = append(diffInos, ino)
			}
		}

		if len(diffInos) > 0 {
			// Sort and deduplicate (inos must be sorted for the ioctl)
			sort.Slice(diffInos, func(i, j int) bool {
				return diffInos[i] < diffInos[j]
			})

			results, err := scoutfs.RawReadInodeInfo(f, diffInos, xattrNames, resultsBuf)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading inode info: %v\n", err)
				os.Exit(1)
			}

			// Build set of returned inos
			returnedInos := make(map[uint64]bool)
			for _, r := range results {
				returnedInos[r.Ino] = true
			}

			// Remove inodes that weren't returned (nlink==0 or deleted)
			for _, ino := range diffInos {
				if !returnedInos[ino] {
					if *debug {
						fmt.Printf("  remove ino %d\n", ino)
					}
					delete(known, ino)
				}
			}

			// Print header periodically
			if lines%16 == 0 {
				fmt.Printf("%8s %12s %12s %12s %6s %6s %6s %6s",
					"ino", "meta_seq", "data_seq", "data_ver",
					"nlink", "uid", "gid", "mode")
				if len(xattrNames) > 0 {
					fmt.Printf(" xattrs")
				}
				fmt.Println()
			}

			for _, r := range results {
				action := "update"
				if _, ok := known[r.Ino]; !ok {
					action = "add"
				}

				known[r.Ino] = &inoState{metaSeq: r.Inode.Meta_seq}

				fmt.Printf("%8d %12d %12d %12d %6d %6d %6d %6o",
					r.Ino,
					r.Inode.Meta_seq,
					r.Inode.Data_seq,
					r.Inode.Data_version,
					r.Inode.Nlink,
					r.Inode.Uid,
					r.Inode.Gid,
					r.Inode.Mode)

				if len(r.Xattrs) > 0 {
					for _, x := range r.Xattrs {
						fmt.Printf(" %s=%q", x.Name, x.Value)
					}
				}

				if *debug {
					fmt.Printf(" [%s]", action)
				}

				fmt.Println()
				lines++
			}
		}

		// Update known state from current scan
		for ino, metaSeq := range seen {
			if st, ok := known[ino]; ok {
				st.metaSeq = metaSeq
			}
		}

		if *debug {
			fmt.Printf("tracking %d inodes, inode result size %d\n",
				len(known), unsafe.Sizeof(scoutfs.ScoutfsInode{}))
		}

		time.Sleep(1 * time.Second)
	}
}

// stringSlice implements flag.Value for repeated string flags
type stringSlice []string

func (s *stringSlice) String() string { return fmt.Sprintf("%v", *s) }
func (s *stringSlice) Set(val string) error {
	*s = append(*s, val)
	return nil
}
