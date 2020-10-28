// Code generated by cmd/cgo -godefs; DO NOT EDIT.
// cgo -godefs c_defs_linux.go

package scoutfs

const IOCQUERYINODES = 0x80487301
const IOCINOPATH = 0x80287302
const IOCRELEASE = 0x40187303
const IOCSTAGE = 0x40207304
const IOCSTATMORE = 0x80307305
const IOCDATAWAITING = 0x80287306
const IOCSETATTRMORE = 0x40287307
const IOCLISTXATTRHIDDEN = 0x80187308
const IOCSEARCHXATTRS = 0x80387309
const IOCSTATFSMORE = 0x8030730a
const IOCDATAWAITERR = 0x8030730b
const IOCALLOCDETAIL = 0x8010730c

const QUERYINODESMETASEQ = 0x0
const QUERYINODESDATASEQ = 0x1

const DATAWAITOPREAD = 0x1
const DATAWAITOPWRITE = 0x2
const DATAWAITOPCHANGESIZE = 0x4

const SEARCHXATTRSOFLAGEND = 0x1

type InodesEntry struct {
	Major	uint64
	Ino	uint64
	Minor	uint32
	X_pad	[4]uint8
}
type queryInodes struct {
	First		InodesEntry
	Last		InodesEntry
	Entries_ptr	uint64
	Nr_entries	uint32
	Index		uint8
	X_pad		[11]uint8
}
type inoPath struct {
	Ino		uint64
	Dir_ino		uint64
	Dir_pos		uint64
	Result_ptr	uint64
	Result_bytes	uint16
	X_pad		[6]uint8
}
type iocRelease struct {
	Block	uint64
	Count	uint64
	Version	uint64
}
type iocStage struct {
	Data_version	uint64
	Buf_ptr		uint64
	Offset		uint64
	Count		int32
	X_pad		uint32
}
type Stat struct {
	Valid_bytes	uint64
	Meta_seq	uint64
	Data_seq	uint64
	Data_version	uint64
	Online_blocks	uint64
	Offline_blocks	uint64
}
type DataWaitingEntry struct {
	Ino	uint64
	Iblock	uint64
	Op	uint8
	X_pad	[7]uint8
}
type dataWaiting struct {
	Flags		uint64
	After_ino	uint64
	After_iblock	uint64
	Ents_ptr	uint64
	Ents_nr		uint16
	X_pad		[6]uint8
}
type dataWaitErr struct {
	Ino	uint64
	Version	uint64
	Offset	uint64
	Count	uint64
	Op	uint64
	Err	int64
}
type setattrMore struct {
	Data_version	uint64
	I_size		uint64
	Flags		uint64
	Ctime_sec	uint64
	Ctime_nsec	uint32
	X_pad		[4]uint8
}
type listXattrHidden struct {
	Id_pos		uint64
	Buf_ptr		uint64
	Buf_bytes	uint32
	Hash_pos	uint32
}
type searchXattrs struct {
	Next_ino	uint64
	Last_ino	uint64
	Name_ptr	uint64
	Inodes_ptr	uint64
	Output_flags	uint64
	Nr_inodes	uint64
	Name_bytes	uint16
	X_pad		[6]uint8
}
type statfsMore struct {
	Valid_bytes		uint64
	Fsid			uint64
	Rid			uint64
	Committed_seq		uint64
	Total_meta_blocks	uint64
	Total_data_blocks	uint64
}
type allocDetail struct {
	Ptr	uint64
	Nr	uint64
}
type allocDetailEntry struct {
	Id		uint64
	Blocks		uint64
	Type		uint8
	Flags		uint8
	Pad_cgo_0	[6]byte
}

const sizeofstatfsMore = 0x30
