// +build ignore

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
// typedef struct scoutfs_ioctl_setattr_more scoutfs_ioctl_setattr_more_t;
// typedef struct scoutfs_ioctl_listxattr_hidden scoutfs_ioctl_listxattr_hidden_t;
// typedef struct scoutfs_ioctl_find_xattrs scoutfs_ioctl_find_xattrs_t;
// typedef struct scoutfs_ioctl_statfs_more scoutfs_ioctl_statfs_more_t;
import "C"

const IOCQUERYINODES = C.SCOUTFS_IOC_WALK_INODES
const IOCINOPATH = C.SCOUTFS_IOC_INO_PATH
const IOCRELEASE = C.SCOUTFS_IOC_RELEASE
const IOCSTAGE = C.SCOUTFS_IOC_STAGE
const IOCSTATMORE = C.SCOUTFS_IOC_STAT_MORE
const IOCDATAWAITING = C.SCOUTFS_IOC_DATA_WAITING
const IOCSETATTRMORE = C.SCOUTFS_IOC_SETATTR_MORE
const IOCLISTXATTRHIDDEN = C.SCOUTFS_IOC_LISTXATTR_HIDDEN
const IOCFINDXATTRS = C.SCOUTFS_IOC_FIND_XATTRS
const IOCSTATFSMORE = C.SCOUTFS_IOC_STATFS_MORE

const QUERYINODESMETASEQ = C.SCOUTFS_IOC_WALK_INODES_META_SEQ
const QUERYINODESDATASEQ = C.SCOUTFS_IOC_WALK_INODES_DATA_SEQ

const DATAWAITOPREAD = C.SCOUTFS_IOC_DWO_READ
const DATAWAITOPWRITE = C.SCOUTFS_IOC_DWO_WRITE
const DATAWAITOPCHANGESIZE = C.SCOUTFS_IOC_DWO_CHANGE_SIZE

type InodesEntry C.scoutfs_ioctl_walk_inodes_entry_t
type queryInodes C.scoutfs_ioctl_walk_inodes_t
type inoPath C.scoutfs_ioctl_ino_path_t
type iocRelease C.scoutfs_ioctl_release_t
type iocStage C.scoutfs_ioctl_stage_t
type Stat C.scoutfs_ioctl_stat_more_t
type DataWaitingEntry C.scoutfs_ioctl_data_waiting_entry_t
type dataWaiting C.scoutfs_ioctl_data_waiting_t
type setattrMore C.scoutfs_ioctl_setattr_more_t
type listXattrHidden C.scoutfs_ioctl_listxattr_hidden_t
type findXattrs C.scoutfs_ioctl_find_xattrs_t
type statfsMore C.scoutfs_ioctl_statfs_more_t

const sizeofstatfsMore = C.sizeof_scoutfs_ioctl_statfs_more_t
