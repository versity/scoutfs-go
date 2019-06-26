// Copyright (c) 2018 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

// The _ struct members are not necessary, but just there to be explicit
// in the matching up with the C struct alignment.  The Go structs follow
// the same alignment rules, so should be compatible without these fields.

package scoutfs

const (
	// IOCQUERYINODES scoutfs ioctl
	IOCQUERYINODES = 0x40487301
	// IOCINOPATH scoutfs ioctl
	IOCINOPATH = 0x40287302
	// IOCDATAVERSION scoutfs ioctl
	IOCDATAVERSION = 0x40087304
	// IOCRELEASE scoutfs ioctl
	IOCRELEASE = 0x40187305
	// IOCSTAGE scoutfs ioctl
	IOCSTAGE = 0x40207306
	// IOCSTATMORE scoutfs ioctl
	IOCSTATMORE = 0x40307307
	// IOCDATAWAITING scoutfs ioctl
	IOCDATAWAITING = 0x40287309
	// IOCSETATTRMORE scoutfs ioctl
	IOCSETATTRMORE = 0x4028730a
	// IOCLISTXATTRRAW scoutfs ioctl
	IOCLISTXATTRRAW = 0x4018730b
	// IOCFINDXATTRS scoutfs ioctl
	IOCFINDXATTRS = 0x4020730c

	// QUERYINODESMETASEQ find inodes by metadata sequence
	QUERYINODESMETASEQ = '\u0000'
	// QUERYINODESDATASEQ find inodes by data sequence
	QUERYINODESDATASEQ = '\u0001'

	// DATAWAITOPREAD waiting operation read
	DATAWAITOPREAD = 1 << 0
	// DATAWAITOPWRITE waiting operation write
	DATAWAITOPWRITE = 1 << 1
	// DATAWAITOPCHANGESIZE waiting operation truncate
	DATAWAITOPCHANGESIZE = 1 << 2

	pathmax = 1024
)

/* pahole for scoutfs_ioctl_walk_inodes_entry
struct scoutfs_ioctl_walk_inodes_entry {
        __u64                      major;                //     0     8
        __u64                      ino;                  //     8     8
        __u32                      minor;                //    16     4
        __u8                       _pad[4];              //    20     4

        // size: 24, cachelines: 1, members: 4
        // last cacheline: 24 bytes
};
*/

// InodesEntry is scoutfs entry for inode iteration
type InodesEntry struct {
	Major uint64
	Ino   uint64
	Minor uint32
	_     [4]uint8
}

/* pahole for scoutfs_ioctl_walk_inodes
struct scoutfs_ioctl_walk_inodes {
        struct scoutfs_ioctl_walk_inodes_entry first;    //     0    24
        struct scoutfs_ioctl_walk_inodes_entry last;     //    24    24
        __u64                      entries_ptr;          //    48     8
        __u32                      nr_entries;           //    56     4
        __u8                       index;                //    60     1
        __u8                       _pad[11];             //    61    11
        // --- cacheline 1 boundary (64 bytes) was 8 bytes ago ---

        // size: 72, cachelines: 2, members: 6
        // last cacheline: 8 bytes
};
*/

// queryInodes is scoutfs request structure for IOCQUERYINODES
type queryInodes struct {
	first   InodesEntry
	last    InodesEntry
	entries uintptr
	count   uint32
	index   uint8
	_       [11]uint8
}

/* pahole for scoutfs_ioctl_ino_path
struct scoutfs_ioctl_ino_path {
        __u64                      ino;                  //     0     8
        __u64                      dir_ino;              //     8     8
        __u64                      dir_pos;              //    16     8
        __u64                      result_ptr;           //    24     8
        __u16                      result_bytes;         //    32     2
        __u8                       _pad[6];              //    34     6

        // size: 40, cachelines: 1, members: 6
        // last cacheline: 40 bytes
};
*/

// InoPath ioctl struct
type InoPath struct {
	Ino        uint64
	DirIno     uint64
	DirPos     uint64
	ResultPtr  uint64
	ResultSize uint16
	_          [6]uint8
}

/* pahole for scoutfs_ioctl_ino_path_result
struct scoutfs_ioctl_ino_path_result {
        __u64                      dir_ino;              //     0     8
        __u64                      dir_pos;              //     8     8
        __u16                      path_bytes;           //    16     2
        __u8                       _pad[6];              //    18     6
        __u8                       path[0];              //    24     0

        // size: 24, cachelines: 1, members: 5
        // last cacheline: 24 bytes
};
*/

// InoPathResult ioctl struct
type InoPathResult struct {
	DirIno   uint64
	DirPos   uint64
	PathSize uint16
	_        [6]uint8
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
        __u32                      _pad;                 //    28     4

        // size: 32, cachelines: 1, members: 5
        // last cacheline: 32 bytes
};
*/

// IocStage ioctl struct
type IocStage struct {
	DataVersion uint64
	BufPtr      uint64
	Offset      uint64
	count       int32
	_           uint32
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

/* pahole for scoutfs_ioctl_data_waiting_entry
struct scoutfs_ioctl_data_waiting_entry {
        __u64                      ino;                  //     0     8
        __u64                      iblock;               //     8     8
        __u8                       op;                   //    16     1
        __u8                       _pad[7];              //    17     7

        // size: 24, cachelines: 1, members: 4
        // last cacheline: 24 bytes
};
*/

// DataWaitingEntry is an entry returned when a process is waiting on
// access of offline block
type DataWaitingEntry struct {
	Ino    uint64
	Iblock uint64
	Op     uint8
	_      [7]uint8
}

/* pahole for scoutfs_ioctl_data_waiting
struct scoutfs_ioctl_data_waiting {
        __u64                      flags;                //     0     8
        __u64                      after_ino;            //     8     8
        __u64                      after_iblock;         //    16     8
        __u64                      ents_ptr;             //    24     8
        __u16                      ents_nr;              //    32     2
        __u8                       _pad[6];              //    34     6

        // size: 40, cachelines: 1, members: 6
        // last cacheline: 40 bytes
};
*/

type dataWaiting struct {
	flags       uint64
	afterIno    uint64
	afterIblock uint64
	entries     uintptr
	count       uint16
	_           [6]uint8
}

/* pahole for scoutfs_ioctl_setattr_more
struct scoutfs_ioctl_setattr_more {
        __u64                      data_version;         //     0     8
        __u64                      i_size;               //     8     8
        __u64                      flags;                //    16     8
        __u64                      ctime_sec;            //    24     8
        __u32                      ctime_nsec;           //    32     4
        __u8                       _pad[4];              //    36     4

        // size: 40, cachelines: 1, members: 6
        // last cacheline: 40 bytes
};
*/

type setattrMore struct {
	dataVersion uint64
	iSize       uint64
	flags       uint64
	ctimesec    uint64
	ctimensec   uint32
	_           [4]uint8
}

/* pahole for scoutfs_ioctl_listxattr_raw
struct scoutfs_ioctl_listxattr_raw {
        __u64                      id_pos;               //     0     8
        __u64                      buf_ptr;              //     8     8
        __u32                      buf_bytes;            //    16     4
        __u32                      hash_pos;             //    20     4

        // size: 24, cachelines: 1, members: 4
        // last cacheline: 24 bytes
};
*/

type listXattrRaw struct {
	idPos   uint64
	buf     uintptr
	bufSize uint32
	hashPos uint32
}

/* pahole for scoutfs_ioctl_find_xattrs
struct scoutfs_ioctl_find_xattrs {
        __u64                      next_ino;             //     0     8
        __u64                      name_ptr;             //     8     8
        __u64                      inodes_ptr;           //    16     8
        __u16                      name_bytes;           //    24     2
        __u16                      nr_inodes;            //    26     2
        __u8                       _pad[4];              //    28     4

        // size: 32, cachelines: 1, members: 6
        // last cacheline: 32 bytes
};
*/

type findXattrs struct {
	nextIno    uint64
	name       uintptr
	inodesBuf  uintptr
	nameSize   uint16
	inodeCount uint16
	_          [4]uint8
}
