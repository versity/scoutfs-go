// Copyright (c) 2018 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

package scoutfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	//IOCQUERYINODES scoutfs ioctl
	IOCQUERYINODES = 0x40357301
	//IOCINOPATH scoutfs ioctl
	IOCINOPATH = 0x40227302
	//IOCDATAVERSION scoutfs ioctl
	IOCDATAVERSION = 0x40087304
	//IOCRELEASE scoutfs ioctl
	IOCRELEASE = 0x40187305
	//IOCSTAGE scoutfs ioctl
	IOCSTAGE = 0x401c7306
	//IOCSTATMORE scoutfs ioctl
	IOCSTATMORE = 0x40307307

	//QUERYINODESMETASEQ find inodes by metadata sequence
	QUERYINODESMETASEQ = '\u0000'
	//QUERYINODESDATASEQ find inodes by data sequence
	QUERYINODESDATASEQ = '\u0001'

	pathmax = 1024
)

/* pahole for scoutfs_ioctl_walk_inodes_entry
struct scoutfs_ioctl_walk_inodes_entry {
        __u64                      major;                //     0     8
        __u32                      minor;                //     8     4
        __u64                      ino;                  //    12     8

        // size: 20, cachelines: 1, members: 3
        // last cacheline: 20 bytes
};
*/

//InodesEntry is scoutfs entry for inode iteration
type InodesEntry struct {
	Major uint64
	Minor uint32
	Ino   uint64
}

/* pahole for scoutfs_ioctl_walk_inodes
struct scoutfs_ioctl_walk_inodes {
        struct scoutfs_ioctl_walk_inodes_entry first;    //     0    20
        struct scoutfs_ioctl_walk_inodes_entry last;     //    20    20
        __u64                      entries_ptr;          //    40     8
        __u32                      nr_entries;           //    48     4
        __u8                       index;                //    52     1

        // size: 53, cachelines: 1, members: 5
        // last cacheline: 53 bytes
};
*/

//queryInodes is scoutfs request structure for IOCQUERYINODES
type queryInodes struct {
	first   InodesEntry
	last    InodesEntry
	entries uintptr
	count   uint32
	index   uint8
}

//packed scoutfs_ioctl_walk_inodes requires packing queryInodes byhand
//the 53 size comes from pahole output above
func (q queryInodes) pack() ([53]byte, error) {
	var pack [53]byte
	var pbuf = bytes.NewBuffer(make([]byte, 0, len(pack)))

	if err := binary.Write(pbuf, binary.LittleEndian, q.first.Major); err != nil {
		return [53]byte{}, fmt.Errorf("pack first.Major: %v", err)
	}
	if err := binary.Write(pbuf, binary.LittleEndian, q.first.Minor); err != nil {
		return [53]byte{}, fmt.Errorf("pack first.Minor: %v", err)
	}
	if err := binary.Write(pbuf, binary.LittleEndian, q.first.Ino); err != nil {
		return [53]byte{}, fmt.Errorf("pack first.Ino: %v", err)
	}
	if err := binary.Write(pbuf, binary.LittleEndian, q.last.Major); err != nil {
		return [53]byte{}, fmt.Errorf("pack last.Major: %v", err)
	}
	if err := binary.Write(pbuf, binary.LittleEndian, q.last.Minor); err != nil {
		return [53]byte{}, fmt.Errorf("pack last.Minor: %v", err)
	}
	if err := binary.Write(pbuf, binary.LittleEndian, q.last.Ino); err != nil {
		return [53]byte{}, fmt.Errorf("pack last.Ino: %v", err)
	}
	if err := binary.Write(pbuf, binary.LittleEndian, uint64(q.entries)); err != nil {
		return [53]byte{}, fmt.Errorf("pack entries: %v", err)
	}
	if err := binary.Write(pbuf, binary.LittleEndian, q.count); err != nil {
		return [53]byte{}, fmt.Errorf("pack count: %v", err)
	}
	if err := binary.Write(pbuf, binary.LittleEndian, q.index); err != nil {
		return [53]byte{}, fmt.Errorf("pack index: %v", err)
	}

	if err := binary.Read(pbuf, binary.LittleEndian, &pack); err != nil {
		return [53]byte{}, fmt.Errorf("read packed: %v", err)
	}

	return pack, nil
}

/* pahole for scoutfs_ioctl_ino_path
struct scoutfs_ioctl_ino_path {
	__u64                      ino;                  //     0     8
	__u64                      dir_ino;              //     8     8
	__u64                      dir_pos;              //    16     8
	__u64                      result_ptr;           //    24     8
	__u16                      result_bytes;         //    32     2

	// size: 34, cachelines: 1, members: 5
	// last cacheline: 34 bytes
};
*/

// InoPath ioctl struct
type InoPath struct {
	Ino        uint64
	DirIno     uint64
	DirPos     uint64
	ResultPtr  uint64
	ResultSize uint16
}

/* pahole for scoutfs_ioctl_ino_path_result
struct scoutfs_ioctl_ino_path_result {
	__u64                      dir_ino;              //     0     8
	__u64                      dir_pos;              //     8     8
	__u16                      path_bytes;           //    16     2
	__u8                       path[0];              //    18     0

	// size: 18, cachelines: 1, members: 4
	// last cacheline: 18 bytes
};
*/

// InoPathResult ioctl struct
type InoPathResult struct {
	DirIno   uint64
	DirPos   uint64
	PathSize uint16
	Path     [pathmax]byte
}

/* pahole for scoutfs_ioctl_release
struct scoutfs_ioctl_release {
	__u64                      block;                //     0     8
	__u64                      count;                //     8     8
	__u64                      data_version;         //    16     8

	// size: 24, cachelines: 1, members: 3
	// last cacheline: 24 bytes
};
*/

// IocRelease ioctl struct
type IocRelease struct {
	Block       uint64
	Count       uint64
	DataVersion uint64
}

/* pahole for scoutfs_ioctl_stage
struct scoutfs_ioctl_stage {
	__u64                      data_version;         //     0     8
	__u64                      buf_ptr;              //     8     8
	__u64                      offset;               //    16     8
	__s32                      count;                //    24     4

	// size: 28, cachelines: 1, members: 4
	// last cacheline: 28 bytes
};
*/

// IocStage ioctl struct
type IocStage struct {
	DataVersion uint64
	BufPtr      uint64
	Offset      uint64
	count       int32
}

/* pahole for scoutfs_ioctl_stat_more
struct scoutfs_ioctl_stat_more {
        __u64                      valid_bytes;          //     0     8
        __u64                      meta_seq;             //     8     8
        __u64                      data_seq;             //    16     8
        __u64                      data_version;         //    24     8
        __u64                      online_blocks;        //    32     8
        __u64                      offline_blocks;       //    40     8

        // size: 48, cachelines: 1, members: 6
        // last cacheline: 48 bytes
};
*/

// Stat holds scoutfs specific per file metadata
type Stat struct {
	ValidBytes    uint64
	MetaSeq       uint64
	DataSeq       uint64
	DataVersion   uint64
	OnlineBlocks  uint64
	OfflineBlocks uint64
}

/* pahole for scoutfs_fid
struct scoutfs_fid {
	__le64                     ino;                  //     0     8
	__le64                     parent_ino;           //     8     8

	// size: 16, cachelines: 1, members: 2
	// last cacheline: 16 bytes
};
*/

// FileID for file by ID operations
type FileID struct {
	Ino       uint64
	ParentIno uint64
}

/* pahole for scoutfs_file_handle
struct scoutfs_file_handle {
	unsigned int               handle_bytes;         //     0     4
	int                        handle_type;          //     4     4
	struct scoutfs_fid         fid;                  //     8    16

	// size: 24, cachelines: 1, members: 3
	// last cacheline: 24 bytes
};
*/

// FileHandle is the scoutfs specific file handle for open by handle operations
type FileHandle struct {
	FidSize    uint32
	HandleType int32
	FID        FileID
}
