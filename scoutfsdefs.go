// Code generated by cmd/cgo -godefs; DO NOT EDIT.
// cgo -godefs c_defs_linux.go

package scoutfs

const IOCQUERYINODES = 0x4048e801
const IOCINOPATH = 0x4028e802
const IOCRELEASE = 0x4018e803
const IOCSTAGE = 0x4020e804
const IOCSTATMORE = 0x8038e805
const IOCDATAWAITING = 0x4028e806
const IOCSETATTRMORE = 0x4030e807
const IOCLISTXATTRHIDDEN = 0xc018e808
const IOCSEARCHXATTRS = 0x4038e809
const IOCSTATFSMORE = 0x8030e80a
const IOCDATAWAITERR = 0x4030e80b
const IOCALLOCDETAIL = 0x4010e80c
const IOCMOVEBLOCKS = 0x4030e80d
const IOCREADXATTRTOTALS = 0x4028e80f
const IOCGETREFERRINGENTRIES = 0x4028e811
const IOCGETQUOTARULES = 0x8020e814
const IOCDELQUOTARULE = 0x4030e816
const IOCADDQUOTARULE = 0x4030e815
const IOCREADXATTRINDEX = 0x8048e817

const QUERYINODESMETASEQ = 0x0
const QUERYINODESDATASEQ = 0x1

const DATAWAITOPREAD = 0x1
const DATAWAITOPWRITE = 0x2
const DATAWAITOPCHANGESIZE = 0x4

const SEARCHXATTRSOFLAGEND = 0x1

const MBSTAGEFLG = 0x1

const DIRENTFLAGLAST = 0x1

const IOCIAXFSIZEOFFLINE = 0x1
const IOCIAXBRETENTION = 0x1
const IOCIAXMETASEQ = 0x1
const IOCIAXDATASEQ = 0x2
const IOCIAXDATAVERSION = 0x4
const IOCIAXONLINEBLOCKS = 0x8
const IOCIAXOFFLINEBLOCKS = 0x10
const IOCIAXCTIME = 0x20
const IOCIAXCRTIME = 0x40
const IOCIAXSIZE = 0x80
const IOCIAXRETENTION = 0x100
const IOCIAXPROJECTID = 0x200
const IOCIAXBITS = 0x100
const IOCGETATTRX = 0x4068e812
const IOCSETATTRX = 0x4068e813

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
	Offset	uint64
	Length	uint64
	Version	uint64
}
type iocStage struct {
	Data_version	uint64
	Buf_ptr		uint64
	Offset		uint64
	Length		int32
	X_pad		uint32
}
type Stat struct {
	Meta_seq	uint64
	Data_seq	uint64
	Data_version	uint64
	Online_blocks	uint64
	Offline_blocks	uint64
	Crtime_sec	uint64
	Crtime_nsec	uint32
	X_pad		[4]uint8
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
	Crtime_nsec	uint32
	Crtime_sec	uint64
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
	Fsid			uint64
	Rid			uint64
	Committed_seq		uint64
	Total_meta_blocks	uint64
	Total_data_blocks	uint64
	Reserved_meta_blocks	uint64
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
type moveBlocks struct {
	From_fd		uint64
	From_off	uint64
	Len		uint64
	To_off		uint64
	Data_version	uint64
	Flags		uint64
}
type readXattrTotals struct {
	Pos_name	[3]uint64
	Totals_ptr	uint64
	Totals_bytes	uint64
}
type xattrTotal struct {
	Name	[3]uint64
	Total	uint64
	Count	uint64
}
type getReferringEntries struct {
	Ino		uint64
	Dir_ino		uint64
	Dir_pos		uint64
	Entries_ptr	uint64
	Entries_bytes	uint64
}
type scoutfsDirent struct {
	Dir_ino		uint64
	Dir_pos		uint64
	Ino		uint64
	Entry_bytes	uint16
	Flags		uint8
	D_type		uint8
	Name_len	uint8
	Name		[3]uint8
}
type quotaRule struct {
	Name_val	[3]uint64
	Limit		uint64
	Prio		uint8
	Op		uint8
	Rule_flags	uint8
	Name_source	[3]uint8
	Name_flags	[3]uint8
	X_pad		[7]uint8
}
type getQuotaRules struct {
	Iterator	[2]uint64
	Ptr		uint64
	Nr		uint64
}
type indexEntry struct {
	Minor	uint64
	Ino	uint64
	Major	uint8
	X_pad	[7]uint8
}
type readXattrIndex struct {
	Flags	uint64
	First	indexEntry
	Last	indexEntry
	Ptr	uint64
	Nr	uint64
}
type inodeAttrX struct {
	X_mask		uint64
	X_flags		uint64
	Meta_seq	uint64
	Data_seq	uint64
	Data_version	uint64
	Online_blocks	uint64
	Offline_blocks	uint64
	Ctime_sec	uint64
	Ctime_nsec	uint32
	Crtime_nsec	uint32
	Crtime_sec	uint64
	Size		uint64
	Bits		uint64
	Project_id	uint64
}

const sizeofstatfsMore = 0x30
const sizeofxattrTotal = 0x28
