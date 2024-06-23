//go:build ignore

// Copyright (c) 2018 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

package scoutfs

// use this to generate the types for scoutfs:
// go tool cgo -godefs c_defs_linux.go >scoutfsdefs.go
// above command requires scoutfs-devel package be installed first

// #include <unistd.h>
// #include <stdio.h>
// #include <stdlib.h>
// #include <sys/types.h>
// #include <sys/stat.h>
// #include <sys/ioctl.h>
// #include <fcntl.h>
// #include <errno.h>
// #include <string.h>
// #include <getopt.h>
// #include <ctype.h>
// #include <stdint.h>
// typedef uint8_t __u8;
// typedef uint16_t __u16;
// typedef uint32_t __u32;
// typedef uint64_t __u64;
// typedef uint16_t __le16;
// typedef uint32_t __le32;
// typedef uint64_t __le64;
// typedef int32_t __s32;
// typedef int64_t __s64;
// #define __packed
// #include "/usr/include/scoutfs/ioctl.h"
// typedef struct scoutfs_ioctl_walk_inodes_entry scoutfs_ioctl_walk_inodes_entry_t;
// typedef struct scoutfs_ioctl_walk_inodes scoutfs_ioctl_walk_inodes_t;
// typedef struct scoutfs_ioctl_ino_path scoutfs_ioctl_ino_path_t;
// typedef struct scoutfs_ioctl_ino_path_result scoutfs_ioctl_ino_path_result_t;
// typedef struct scoutfs_ioctl_release scoutfs_ioctl_release_t;
// typedef struct scoutfs_ioctl_stage scoutfs_ioctl_stage_t;
// typedef struct scoutfs_ioctl_stat_more scoutfs_ioctl_stat_more_t;
// typedef struct scoutfs_fid scoutfs_fid_t;
// typedef struct scoutfs_ioctl_data_waiting_entry scoutfs_ioctl_data_waiting_entry_t;
// typedef struct scoutfs_ioctl_data_waiting scoutfs_ioctl_data_waiting_t;
// typedef struct scoutfs_ioctl_data_wait_err scoutfs_ioctl_data_wait_err_t;
// typedef struct scoutfs_ioctl_setattr_more scoutfs_ioctl_setattr_more_t;
// typedef struct scoutfs_ioctl_listxattr_hidden scoutfs_ioctl_listxattr_hidden_t;
// typedef struct scoutfs_ioctl_search_xattrs scoutfs_ioctl_search_xattrs_t;
// typedef struct scoutfs_ioctl_statfs_more scoutfs_ioctl_statfs_more_t;
// typedef struct scoutfs_ioctl_alloc_detail scoutfs_ioctl_alloc_detail_t;
// typedef struct scoutfs_ioctl_read_xattr_totals scoutfs_ioctl_read_xattr_totals_t;
// typedef struct scoutfs_ioctl_xattr_total scoutfs_ioctl_xattr_total_t;
// typedef struct scoutfs_ioctl_get_referring_entries scoutfs_ioctl_get_referring_entries_t;
// typedef struct scoutfs_ioctl_dirent scoutfs_ioctl_dirent_t;
//
// // Go doesnt handle bitfields in structs, so we need to override the scoutfs
// // struct definition here
// struct scoutfs_ioctl_alloc_detail_entry_mod {
// 	__u64 id;
// 	__u64 blocks;
// 	__u8 type;
// 	__u8 flags;
// };
//
// typedef struct scoutfs_ioctl_alloc_detail_entry_mod scoutfs_ioctl_alloc_detail_entry_t;
// typedef struct scoutfs_ioctl_move_blocks scoutfs_ioctl_move_blocks_t;
// typedef struct scoutfs_ioctl_quota_rule scoutfs_ioctl_quota_rule_t;
// typedef struct scoutfs_ioctl_get_quota_rules scoutfs_ioctl_get_quota_rules_t;
// typedef struct scoutfs_ioctl_xattr_index_entry scoutfs_ioctl_xattr_index_entry_t;
// typedef struct scoutfs_ioctl_read_xattr_index scoutfs_ioctl_read_xattr_index_t;
// typedef struct scoutfs_ioctl_inode_attr_x scoutfs_ioctl_inode_attr_x_t;
import "C"

const IOCQUERYINODES = C.SCOUTFS_IOC_WALK_INODES
const IOCINOPATH = C.SCOUTFS_IOC_INO_PATH
const IOCRELEASE = C.SCOUTFS_IOC_RELEASE
const IOCSTAGE = C.SCOUTFS_IOC_STAGE
const IOCSTATMORE = C.SCOUTFS_IOC_STAT_MORE
const IOCDATAWAITING = C.SCOUTFS_IOC_DATA_WAITING
const IOCSETATTRMORE = C.SCOUTFS_IOC_SETATTR_MORE
const IOCLISTXATTRHIDDEN = C.SCOUTFS_IOC_LISTXATTR_HIDDEN
const IOCSEARCHXATTRS = C.SCOUTFS_IOC_SEARCH_XATTRS
const IOCSTATFSMORE = C.SCOUTFS_IOC_STATFS_MORE
const IOCDATAWAITERR = C.SCOUTFS_IOC_DATA_WAIT_ERR
const IOCALLOCDETAIL = C.SCOUTFS_IOC_ALLOC_DETAIL
const IOCMOVEBLOCKS = C.SCOUTFS_IOC_MOVE_BLOCKS
const IOCREADXATTRTOTALS = C.SCOUTFS_IOC_READ_XATTR_TOTALS
const IOCGETREFERRINGENTRIES = C.SCOUTFS_IOC_GET_REFERRING_ENTRIES
const IOCGETQUOTARULES = C.SCOUTFS_IOC_GET_QUOTA_RULES
const IOCDELQUOTARULE = C.SCOUTFS_IOC_DEL_QUOTA_RULE
const IOCADDQUOTARULE = C.SCOUTFS_IOC_ADD_QUOTA_RULE
const IOCREADXATTRINDEX = C.SCOUTFS_IOC_READ_XATTR_INDEX

const QUERYINODESMETASEQ = C.SCOUTFS_IOC_WALK_INODES_META_SEQ
const QUERYINODESDATASEQ = C.SCOUTFS_IOC_WALK_INODES_DATA_SEQ

const DATAWAITOPREAD = C.SCOUTFS_IOC_DWO_READ
const DATAWAITOPWRITE = C.SCOUTFS_IOC_DWO_WRITE
const DATAWAITOPCHANGESIZE = C.SCOUTFS_IOC_DWO_CHANGE_SIZE

const SEARCHXATTRSOFLAGEND = C.SCOUTFS_SEARCH_XATTRS_OFLAG_END

const MBSTAGEFLG = C.SCOUTFS_IOC_MB_STAGE

const DIRENTFLAGLAST = C.SCOUTFS_IOCTL_DIRENT_FLAG_LAST

const IOCIAXFSIZEOFFLINE = C.SCOUTFS_IOC_IAX_F_SIZE_OFFLINE
const IOCIAXBRETENTION = C.SCOUTFS_IOC_IAX_B_RETENTION
const IOCIAXMETASEQ = C.SCOUTFS_IOC_IAX_META_SEQ
const IOCIAXDATASEQ = C.SCOUTFS_IOC_IAX_DATA_SEQ
const IOCIAXDATAVERSION = C.SCOUTFS_IOC_IAX_DATA_VERSION
const IOCIAXONLINEBLOCKS = C.SCOUTFS_IOC_IAX_ONLINE_BLOCKS
const IOCIAXOFFLINEBLOCKS = C.SCOUTFS_IOC_IAX_OFFLINE_BLOCKS
const IOCIAXCTIME = C.SCOUTFS_IOC_IAX_CTIME
const IOCIAXCRTIME = C.SCOUTFS_IOC_IAX_CRTIME
const IOCIAXSIZE = C.SCOUTFS_IOC_IAX_SIZE
const IOCIAXRETENTION = C.SCOUTFS_IOC_IAX_RETENTION
const IOCIAXPROJECTID = C.SCOUTFS_IOC_IAX_PROJECT_ID
const IOCIAXBITS = C.SCOUTFS_IOC_IAX__BITS
const IOCGETATTRX = C.SCOUTFS_IOC_GET_ATTR_X
const IOCSETATTRX = C.SCOUTFS_IOC_SET_ATTR_X

type InodesEntry C.scoutfs_ioctl_walk_inodes_entry_t
type queryInodes C.scoutfs_ioctl_walk_inodes_t
type inoPath C.scoutfs_ioctl_ino_path_t
type iocRelease C.scoutfs_ioctl_release_t
type iocStage C.scoutfs_ioctl_stage_t
type Stat C.scoutfs_ioctl_stat_more_t
type DataWaitingEntry C.scoutfs_ioctl_data_waiting_entry_t
type dataWaiting C.scoutfs_ioctl_data_waiting_t
type dataWaitErr C.scoutfs_ioctl_data_wait_err_t
type setattrMore C.scoutfs_ioctl_setattr_more_t
type listXattrHidden C.scoutfs_ioctl_listxattr_hidden_t
type searchXattrs C.scoutfs_ioctl_search_xattrs_t
type statfsMore C.scoutfs_ioctl_statfs_more_t
type allocDetail C.scoutfs_ioctl_alloc_detail_t
type allocDetailEntry C.scoutfs_ioctl_alloc_detail_entry_t
type moveBlocks C.scoutfs_ioctl_move_blocks_t
type readXattrTotals C.scoutfs_ioctl_read_xattr_totals_t
type xattrTotal C.scoutfs_ioctl_xattr_total_t
type getReferringEntries C.scoutfs_ioctl_get_referring_entries_t
type scoutfsDirent C.scoutfs_ioctl_dirent_t
type quotaRule C.scoutfs_ioctl_quota_rule_t
type getQuotaRules C.scoutfs_ioctl_get_quota_rules_t
type indexEntry C.scoutfs_ioctl_xattr_index_entry_t
type readXattrIndex C.scoutfs_ioctl_read_xattr_index_t
type inodeAttrX C.scoutfs_ioctl_inode_attr_x_t

const sizeofstatfsMore = C.sizeof_scoutfs_ioctl_statfs_more_t
const sizeofxattrTotal = C.sizeof_scoutfs_ioctl_xattr_total_t
