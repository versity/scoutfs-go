// Copyright (c) 2018 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

package scoutfs

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	max64            = 0xffffffffffffffff
	max32            = 0xffffffff
	pathmax          = 4096
	sysscoutfs       = "/sys/fs/scoutfs/"
	statusfile       = "quorum/status"
	listattrBufsize  = 256 * 1024
	getparentBufsize = 4096 * 1024
	scoutfsBS        = 4096
	//leaderfile      = "quorum/is_leader"
)

// Query to keep track of in-process query
type Query struct {
	first InodesEntry
	last  InodesEntry
	index uint8
	batch uint32
	fsfd  *os.File
	buf   []byte
}

// Time represents a time value in seconds and nanoseconds
type Time struct {
	Sec  uint64
	Nsec uint32
}

// NewQuery creates a new scoutfs Query
// Specify query type with By*() option
// (only 1 allowed, last one wins)
// and specify batching with WithBatchSize()
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func NewQuery(f *os.File, opts ...Option) *Query {
	q := &Query{
		//default batch size is 128
		batch: 128,
		fsfd:  f,
	}

	for _, opt := range opts {
		opt(q)
	}

	q.buf = make([]byte, int(unsafe.Sizeof(InodesEntry{}))*int(q.batch))

	return q
}

// Option sets various options for NewQuery
type Option func(*Query)

// ByMSeq gets inodes in range of metadata sequence from, to inclusive
func ByMSeq(from, to InodesEntry) Option {
	return func(q *Query) {
		q.first = from
		q.last = to
		q.index = QUERYINODESMETASEQ
	}
}

// ByDSeq gets inodes in range of data sequence from, to inclusive
func ByDSeq(from, to InodesEntry) Option {
	return func(q *Query) {
		q.first = from
		q.last = to
		q.index = QUERYINODESDATASEQ
	}
}

// WithBatchSize sets the max number of inodes to be returned at a time
func WithBatchSize(size uint32) Option {
	return func(q *Query) {
		q.batch = size
	}
}

// Next gets the next batch of inodes
func (q *Query) Next() ([]InodesEntry, error) {
	query := queryInodes{
		First:       q.first,
		Last:        q.last,
		Entries_ptr: uint64(uintptr(unsafe.Pointer(&q.buf[0]))),
		Nr_entries:  q.batch,
		Index:       q.index,
	}

	n, err := scoutfsctl(q.fsfd, IOCQUERYINODES, unsafe.Pointer(&query))
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	rbuf := bytes.NewReader(q.buf)
	inodes := make([]InodesEntry, n)

	var e InodesEntry
	for i := 0; i < n; i++ {
		err := binary.Read(rbuf, binary.LittleEndian, &e)
		if err != nil {
			return nil, err
		}

		inodes[i] = e
	}

	q.first = e.Increment()
	return inodes, nil
}

// SetLast updates the sequence stopping point
func (q *Query) SetLast(l InodesEntry) {
	q.last = l
}

// Increment returns the next seq entry position
func (i InodesEntry) Increment() InodesEntry {
	i.Ino++
	if i.Ino == 0 {
		i.Minor++
		if i.Minor == 0 {
			i.Major++
		}
	}
	return i
}

// String returns the string representation of InodesEntry
func (i InodesEntry) String() string {
	return fmt.Sprintf("{seq: %v, ino: %v}", i.Major, i.Ino)
}

// StatMore returns scoutfs specific metadata for path
func StatMore(path string) (Stat, error) {
	f, err := os.Open(path)
	if err != nil {
		return Stat{}, err
	}
	defer f.Close()

	return FStatMore(f)
}

// FStatMore returns scoutfs specific metadata for file handle
func FStatMore(f *os.File) (Stat, error) {
	s := Stat{}

	_, err := scoutfsctl(f, IOCSTATMORE, unsafe.Pointer(&s))
	if err != nil {
		return Stat{}, err
	}

	return s, nil
}

// SetAttrMore sets special scoutfs attributes
func SetAttrMore(path string, version, size, flags uint64, ctime time.Time, crtime time.Time) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return FSetAttrMore(f, version, size, flags, ctime, crtime)
}

// FSetAttrMore sets special scoutfs attributes for file handle
func FSetAttrMore(f *os.File, version, size, flags uint64, ctime time.Time, crtime time.Time) error {
	var cnsec int32
	var crnsec int32
	if ctime.Nanosecond() == int(int32(ctime.Nanosecond())) {
		cnsec = int32(ctime.Nanosecond())
	}
	if crtime.Nanosecond() == int(int32(crtime.Nanosecond())) {
		crnsec = int32(crtime.Nanosecond())
	}
	s := setattrMore{
		Data_version: version,
		I_size:       size,
		Flags:        flags,
		Ctime_sec:    uint64(ctime.Unix()),
		Ctime_nsec:   uint32(cnsec),
		Crtime_sec:   uint64(crtime.Unix()),
		Crtime_nsec:  uint32(crnsec),
	}

	_, err := scoutfsctl(f, IOCSETATTRMORE, unsafe.Pointer(&s))
	return err
}

type inoPathResult struct {
	DirIno   uint64
	DirPos   uint64
	PathSize uint16
	_        [6]uint8
	Path     [pathmax]byte
}

// InoToPath converts an inode number to a path in the filesystem
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func InoToPath(dirfd *os.File, ino uint64) (string, error) {
	var res inoPathResult
	ip := inoPath{
		Ino:          ino,
		Result_ptr:   uint64(uintptr(unsafe.Pointer(&res))),
		Result_bytes: uint16(unsafe.Sizeof(res)),
	}

	_, err := scoutfsctl(dirfd, IOCINOPATH, unsafe.Pointer(&ip))
	if err != nil {
		return "", err
	}

	b := bytes.Trim(res.Path[:res.PathSize], "\x00")

	return string(b), nil
}

// InoToPaths converts an inode number to all paths in the filesystem
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func InoToPaths(dirfd *os.File, ino uint64) ([]string, error) {
	var res inoPathResult
	ip := inoPath{
		Ino:          ino,
		Result_ptr:   uint64(uintptr(unsafe.Pointer(&res))),
		Result_bytes: uint16(unsafe.Sizeof(res)),
	}

	var paths []string
	for {
		_, err := scoutfsctl(dirfd, IOCINOPATH, unsafe.Pointer(&ip))
		if err == syscall.ENOENT {
			break
		}
		if err != nil {
			return nil, err
		}

		b := bytes.Trim(res.Path[:res.PathSize], "\x00")
		paths = append(paths, string(b))

		ip.Dir_ino = res.DirIno
		ip.Dir_pos = res.DirPos
		ip.Dir_pos++
		if ip.Dir_pos == 0 {
			ip.Dir_ino++
			if ip.Dir_ino == 0 {
				break
			}
		}
	}

	return paths, nil
}

// OpenByID will open a file by inode returning a typical *os.File
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
// The filename supplied is used for the *os.File info, but can be "" if
// not known or needed
func OpenByID(dirfd *os.File, ino uint64, flags int, name string) (*os.File, error) {
	fd, err := OpenByHandle(dirfd, ino, flags)
	if err != nil {
		return nil, err
	}

	return os.NewFile(fd, name), nil
}

// ReleaseFile sets file offline by freeing associated extents
func ReleaseFile(path string, version uint64) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	return FReleaseFile(f, version)
}

// FReleaseFile set file offline by freeing associated extents
func FReleaseFile(f *os.File, version uint64) error {
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	r := iocRelease{
		Length:  divRoundUp(uint64(fi.Size()), scoutfsBS),
		Version: version,
	}

	_, err = scoutfsctl(f, IOCRELEASE, unsafe.Pointer(&r))
	return err
}

func divRoundUp(size, bs uint64) uint64 {
	d := (size / bs) * bs
	if d == size {
		return d
	}
	return d + bs
}

// ReleaseBlocks marks blocks offline and frees associated extents
// offset/length must be 4k aligned
func ReleaseBlocks(path string, offset, length, version uint64) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	return FReleaseBlocks(f, offset, length, version)
}

// FReleaseBlocks marks blocks offline and frees associated extents
// offset/length must be 4k aligned
func FReleaseBlocks(f *os.File, offset, length, version uint64) error {
	r := iocRelease{
		Offset:  offset,
		Length:  length,
		Version: version,
	}

	_, err := scoutfsctl(f, IOCRELEASE, unsafe.Pointer(&r))
	return err
}

// StageFile rehydrates offline file
func StageFile(path string, version, offset uint64, b []byte) (int, error) {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return FStageFile(f, version, offset, b)
}

// FStageFile rehydrates offline file
func FStageFile(f *os.File, version, offset uint64, b []byte) (int, error) {
	r := iocStage{
		Data_version: version,
		Buf_ptr:      uint64(uintptr(unsafe.Pointer(&b[0]))),
		Offset:       offset,
		Length:       int32(len(b)),
	}

	return scoutfsctl(f, IOCSTAGE, unsafe.Pointer(&r))
}

// Waiters to keep track of data waiters
type Waiters struct {
	ino    uint64
	iblock uint64
	batch  uint16
	fsfd   *os.File
	buf    []byte
}

// NewWaiters creates a new scoutfs Waiters
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func NewWaiters(f *os.File, opts ...WOption) *Waiters {
	w := &Waiters{
		//default batch size is 128
		batch: 128,
		fsfd:  f,
	}

	for _, opt := range opts {
		opt(w)
	}

	w.buf = make([]byte, int(unsafe.Sizeof(DataWaitingEntry{}))*int(w.batch))

	return w
}

// WOption sets various options for NewWaiters
type WOption func(*Waiters)

// WithWaitersCount sets the max number of inodes to be returned at a time
func WithWaitersCount(size uint16) WOption {
	return func(w *Waiters) {
		w.batch = size
	}
}

// Next gets the next batch of data waiters, returns nil, nil if no waiters
func (w *Waiters) Next() ([]DataWaitingEntry, error) {
	dataWaiting := dataWaiting{
		After_ino:    w.ino,
		After_iblock: w.iblock,
		Ents_ptr:     uint64(uintptr(unsafe.Pointer(&w.buf[0]))),
		Ents_nr:      w.batch,
	}

	n, err := scoutfsctl(w.fsfd, IOCDATAWAITING, unsafe.Pointer(&dataWaiting))
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	rbuf := bytes.NewReader(w.buf)
	inodes := make([]DataWaitingEntry, n)

	var e DataWaitingEntry
	for i := 0; i < n; i++ {
		err := binary.Read(rbuf, binary.LittleEndian, &e)
		if err != nil {
			return nil, err
		}

		inodes[i] = e
	}

	w.ino = inodes[n-1].Ino
	w.iblock = inodes[n-1].Iblock

	return inodes, nil
}

// Reset sets the data waiters query back to inode 0, iblock 0
func (w *Waiters) Reset() {
	w.ino = 0
	w.iblock = 0
}

// SendDataWaitErr sends an error to the data waiter task indicating that
// the data is no longer aviable.
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func SendDataWaitErr(dirfd *os.File, ino, version, offset, op, count uint64, errno int64) error {
	derr := dataWaitErr{
		Ino:     ino,
		Version: version,
		Offset:  offset,
		Count:   count,
		Op:      op,
		Err:     errno,
	}

	_, err := scoutfsctl(dirfd, IOCDATAWAITERR, unsafe.Pointer(&derr))
	if err != nil {
		return err
	}
	return nil
}

// XattrQuery to keep track of in-process xattr query
type XattrQuery struct {
	next  uint64
	batch uint64
	key   string
	fsfd  *os.File
	buf   []byte
	done  bool
}

// NewXattrQuery creates a new scoutfs Xattr Query
// Specify query xattr key
// and specify optinally batching with WithXBatchSize()
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func NewXattrQuery(f *os.File, key string, opts ...XOption) *XattrQuery {
	q := &XattrQuery{
		// default batch size is 131072 for a nice round 1MB allocation.
		// making this too small risks multiple calls into Next() wich
		// has significant overhead per call.
		batch: (128 * 1024),
		key:   key,
		fsfd:  f,
	}

	for _, opt := range opts {
		opt(q)
	}

	q.buf = make([]byte, 8*int(q.batch))

	return q
}

// XOption sets various options for NewXattrQuery
type XOption func(*XattrQuery)

// WithXBatchSize sets the max number of inodes to be returned at a time
func WithXBatchSize(size uint64) XOption {
	return func(q *XattrQuery) {
		q.batch = size
	}
}

// WithXStartIno starts query at speficied inode
func WithXStartIno(ino uint64) XOption {
	return func(q *XattrQuery) {
		q.next = ino
	}
}

// Next gets the next batch of inodes
func (q *XattrQuery) Next() ([]uint64, error) {
	name := []byte(q.key)
	query := searchXattrs{
		Next_ino:   q.next,
		Last_ino:   max64,
		Name_ptr:   uint64(uintptr(unsafe.Pointer(&name[0]))),
		Inodes_ptr: uint64(uintptr(unsafe.Pointer(&q.buf[0]))),
		Name_bytes: uint16(len(name)),
		Nr_inodes:  q.batch,
	}

	if q.done {
		return nil, nil
	}

	n, err := scoutfsctl(q.fsfd, IOCSEARCHXATTRS, unsafe.Pointer(&query))
	if err != nil {
		return nil, err
	}

	if query.Output_flags&SEARCHXATTRSOFLAGEND != 0 {
		q.done = true
	}

	if n == 0 {
		return nil, nil
	}

	rbuf := bytes.NewReader(q.buf)
	inodes := make([]uint64, n)

	var e uint64
	for i := 0; i < n; i++ {
		err := binary.Read(rbuf, binary.LittleEndian, &e)
		if err != nil {
			return nil, err
		}

		inodes[i] = e
	}

	q.next = e
	q.next++

	return inodes, nil
}

// ListXattrHidden holds info for iterating on xattrs
type ListXattrHidden struct {
	lxr *listXattrHidden
	f   *os.File
	buf []byte
}

// NewListXattrHidden will list all scoutfs xattrs (including hidden) for file.
// If passed in buffer is nil, call will allocate its own buffer.
func NewListXattrHidden(f *os.File, b []byte) *ListXattrHidden {
	if b == nil {
		b = make([]byte, listattrBufsize)
	}
	return &ListXattrHidden{
		f:   f,
		lxr: &listXattrHidden{},
		buf: b,
	}
}

// Next gets next set of results, complete when string slice is nil
func (l *ListXattrHidden) Next() ([]string, error) {
	l.lxr.Buf_bytes = uint32(len(l.buf))
	l.lxr.Buf_ptr = uint64(uintptr(unsafe.Pointer(&l.buf[0])))

	n, err := scoutfsctl(l.f, IOCLISTXATTRHIDDEN, unsafe.Pointer(l.lxr))
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	return bufToStrings(l.buf[:n]), nil
}

func bufToStrings(b []byte) []string {
	var s []string
	for {
		i := bytes.IndexByte(b, byte(0))
		if i == -1 {
			break
		}
		s = append(s, string(b[0:i]))
		b = b[i+1:]
	}
	return s
}

// FSID contains the statfs more info for mounted scoutfs filesystem
type FSID struct {
	FSID         uint64
	RandomID     uint64
	ShortID      string
	CommittedSeq uint64
}

// GetIDs gets the statfs more filesystem and random id from file handle within
// scoutfs filesystem
func GetIDs(f *os.File) (FSID, error) {
	stfs := statfsMore{}

	_, err := scoutfsctl(f, IOCSTATFSMORE, unsafe.Pointer(&stfs))
	if err != nil {
		return FSID{}, fmt.Errorf("statfs more: %v", err)
	}

	short := fmt.Sprintf("f.%v.r.%v",
		fmt.Sprintf("%016x", stfs.Fsid)[:][:6], fmt.Sprintf("%016x", stfs.Rid)[:][:6])

	return FSID{
		FSID:         stfs.Fsid,
		RandomID:     stfs.Rid,
		ShortID:      short,
		CommittedSeq: stfs.Committed_seq,
	}, nil
}

// QuorumInfo holds info for current mount quorum
type QuorumInfo struct {
	Slot int64
	Term int64
	Role string
}

// IsLeader returns true if quorum status is a leader role
func (q QuorumInfo) IsLeader() bool {
	return q.Role == "(leader)"
}

// GetQuorumInfo returns quorum info for curren mount
func GetQuorumInfo(path string) (QuorumInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return QuorumInfo{}, fmt.Errorf("open: %v", err)
	}
	defer f.Close()

	id, err := GetIDs(f)
	if err != nil {
		return QuorumInfo{}, fmt.Errorf("error GetIDs: %v", err)
	}

	sfspath := filepath.Join(sysscoutfs, id.ShortID, statusfile)
	sfs, err := os.Open(sfspath)
	if err != nil {
		return QuorumInfo{}, fmt.Errorf("open %q: %v", sfspath, err)
	}
	defer sfs.Close()

	qi := QuorumInfo{}
	scanner := bufio.NewScanner(sfs)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			return QuorumInfo{}, fmt.Errorf("parse line (%q): %q",
				sfspath, scanner.Text())
		}
		switch fields[0] {
		case "quorum_slot_nr":
			qi.Slot, err = strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return QuorumInfo{}, fmt.Errorf("parse quorum_slot_nr %q: %v",
					fields[1], err)
			}
		case "term":
			qi.Term, err = strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return QuorumInfo{}, fmt.Errorf("term %q: %v",
					fields[1], err)
			}
		case "role":
			if len(fields) < 3 {
				return QuorumInfo{}, fmt.Errorf("parse line (%q): %q",
					sfspath, scanner.Text())
			}
			qi.Role = fields[2]
		}
	}

	if err := scanner.Err(); err != nil {
		return QuorumInfo{}, fmt.Errorf("parse %q: %v", sfspath, err)
	}

	return qi, nil
}

// DiskUsage holds usage information reported by the filesystem
type DiskUsage struct {
	TotalMetaBlocks uint64
	FreeMetaBlocks  uint64
	TotalDataBlocks uint64
	FreeDataBlocks  uint64
}

var dfBatchCount uint64 = 4096
var metaFlag uint8 = 0x1

// GetDF returns usage data for the filesystem
func GetDF(f *os.File) (DiskUsage, error) {
	stfs := statfsMore{}

	_, err := scoutfsctl(f, IOCSTATFSMORE, unsafe.Pointer(&stfs))
	if err != nil {
		return DiskUsage{}, fmt.Errorf("statfs more: %v", err)
	}

	nr := dfBatchCount
	buf := make([]byte, int(unsafe.Sizeof(allocDetailEntry{}))*int(nr))
	var ret int
	for {
		ad := allocDetail{
			Nr:  nr,
			Ptr: uint64(uintptr(unsafe.Pointer(&buf[0]))),
		}
		ret, err = scoutfsctl(f, IOCALLOCDETAIL, unsafe.Pointer(&ad))
		if err == syscall.EOVERFLOW {
			nr = nr * 2
			buf = make([]byte, int(unsafe.Sizeof(allocDetailEntry{}))*int(nr))
			continue
		}
		if err != nil {
			return DiskUsage{}, fmt.Errorf("alloc detail: %v", err)
		}
		break
	}

	rbuf := bytes.NewReader(buf)
	var ade allocDetailEntry
	var metaFree, dataFree uint64
	for i := 0; i < ret; i++ {
		err := binary.Read(rbuf, binary.LittleEndian, &ade)
		if err != nil {
			return DiskUsage{}, fmt.Errorf("parse alloc detail results: %v", err)
		}
		if ade.Flags&metaFlag != 0 {
			metaFree += ade.Blocks
		} else {
			dataFree += ade.Blocks
		}
	}

	return DiskUsage{
		TotalMetaBlocks: stfs.Total_meta_blocks,
		FreeMetaBlocks:  metaFree,
		TotalDataBlocks: stfs.Total_data_blocks,
		FreeDataBlocks:  dataFree,
	}, nil
}

// MoveData will move all of the extents in "from" file handle
// and append to the end of "to" file handle.
// The end of "to" must be 4KB aligned boundary.
// errors this can return:
// EINVAL: from_off, len, or to_off aren't a multiple of 4KB; the source
//
//	and destination files are the same inode; either the source or
//	destination is not a regular file; the destination file has
//	an existing overlapping extent.
//
// EOVERFLOW: either from_off + len or to_off + len exceeded 64bits.
// EBADF: from_fd isn't a valid open file descriptor.
// EXDEV: the source and destination files are in different filesystems.
// EISDIR: either the source or destination is a directory.
// ENODATA: either the source or destination file have offline extents.
func MoveData(from, to *os.File) error {
	ffi, err := from.Stat()
	if err != nil {
		return fmt.Errorf("stat from: %v", err)
	}
	tfi, err := to.Stat()
	if err != nil {
		return fmt.Errorf("stat to: %v", err)
	}

	mb := moveBlocks{
		From_fd:  uint64(from.Fd()),
		From_off: 0,
		Len:      uint64(ffi.Size()),
		To_off:   uint64(tfi.Size()),
	}

	_, err = scoutfsctl(to, IOCMOVEBLOCKS, unsafe.Pointer(&mb))
	if err != nil {
		return err
	}

	from.Truncate(0)
	from.Seek(0, io.SeekStart)

	return nil
}

// StageMove will move all of the extents in "from" file handle
// and stage the offline extents at offset "offset" in "to" file handle.
// The size of from and offset of "to" must be 4KB aligned boundary.
// errors this can return:
// EINVAL: from_off, len, or to_off aren't a multiple of 4KB; the source
//
//	and destination files are the same inode; either the source or
//	destination is not a regular file; the destination file has
//	an existing overlapping extent.
//
// EOVERFLOW: either from_off + len or to_off + len exceeded 64bits.
// EBADF: from_fd isn't a valid open file descriptor.
// EXDEV: the source and destination files are in different filesystems.
// EISDIR: either the source or destination is a directory.
// ENODATA: either the source or destination file have offline extents.
func StageMove(from, to *os.File, offset, version uint64) error {
	ffi, err := from.Stat()
	if err != nil {
		return fmt.Errorf("stat from: %v", err)
	}

	mb := moveBlocks{
		From_fd:      uint64(from.Fd()),
		From_off:     0,
		Len:          uint64(ffi.Size()),
		To_off:       offset,
		Data_version: version,
		Flags:        MBSTAGEFLG,
	}

	_, err = scoutfsctl(to, IOCMOVEBLOCKS, unsafe.Pointer(&mb))
	if err != nil {
		return err
	}

	from.Truncate(0)
	from.Seek(0, io.SeekStart)

	return nil
}

// StageMoveAt will move the extents (based on len) in "from" file handle at
// given offset to the "to" file handle at given offset.
// All offsets must be 4KB aligned boundary.
// All destination offsets must be offline extents.
// EINVAL: from_off, len, or to_off aren't a multiple of 4KB; the source
//
//	and destination files are the same inode; either the source or
//	destination is not a regular file; the destination file has
//	an existing overlapping extent.
//
// EOVERFLOW: either from_off + len or to_off + len exceeded 64bits.
// EBADF: from_fd isn't a valid open file descriptor.
// EXDEV: the source and destination files are in different filesystems.
// EISDIR: either the source or destination is a directory.
// ENODATA: either the source or destination file have offline extents.
func StageMoveAt(from, to *os.File, len, fromOffset, toOffset, version uint64) error {
	mb := moveBlocks{
		From_fd:      uint64(from.Fd()),
		From_off:     fromOffset,
		Len:          len,
		To_off:       toOffset,
		Data_version: version,
		Flags:        MBSTAGEFLG,
	}

	_, err := scoutfsctl(to, IOCMOVEBLOCKS, unsafe.Pointer(&mb))
	if err != nil {
		return err
	}

	return nil
}

// XattrTotal has the total values matching id triple
type XattrTotal struct {
	// Total is sum of all xattr values matching ids
	Total uint64
	// Count is number of xattrs matching ids
	Count uint64
	// ID is the id for this total
	ID [3]uint64
}

// ReadXattrTotals returns the XattrTotal for the given id
func ReadXattrTotals(f *os.File, id1, id2, id3 uint64) (XattrTotal, error) {
	totls := make([]xattrTotal, 1)

	query := readXattrTotals{
		Pos_name:     [3]uint64{id1, id2, id3},
		Totals_ptr:   uint64(uintptr(unsafe.Pointer(&totls[0]))),
		Totals_bytes: sizeofxattrTotal,
	}

	n, err := scoutfsctl(f, IOCREADXATTRTOTALS, unsafe.Pointer(&query))
	if err != nil {
		return XattrTotal{}, err
	}
	if n == 0 ||
		totls[0].Name[0] != id1 ||
		totls[0].Name[1] != id2 ||
		totls[0].Name[2] != id3 {
		return XattrTotal{}, nil
	}

	return XattrTotal{
		Total: totls[0].Total,
		Count: totls[0].Count,
	}, nil
}

type TotalsGroup struct {
	totls []xattrTotal
	pos   [3]uint64
	id1   uint64
	id2   uint64
	count int
	f     *os.File
	done  bool
}

// NewTotalsGroup creates a query to get the totals values for a defined
// group of totls (group is defined to match first 2 identifiers).  Count
// specifies max number returned for each Next() call.
func NewTotalsGroup(f *os.File, id1, id2 uint64, count int) *TotalsGroup {
	totls := make([]xattrTotal, count)

	return &TotalsGroup{
		totls: totls,
		f:     f,
		id1:   id1,
		id2:   id2,
		count: count,
		pos:   [3]uint64{id1, id2, 0},
	}
}

// Next returns next set of total values for the group
func (t *TotalsGroup) Next() ([]XattrTotal, error) {
	if t.done {
		return nil, nil
	}

	query := readXattrTotals{
		Pos_name:     t.pos,
		Totals_ptr:   uint64(uintptr(unsafe.Pointer(&t.totls[0]))),
		Totals_bytes: sizeofxattrTotal * uint64(t.count),
	}

	n, err := scoutfsctl(t.f, IOCREADXATTRTOTALS, unsafe.Pointer(&query))
	if err != nil {
		return nil, err
	}
	if n == 0 {
		t.done = true
		return nil, nil
	}

	t.pos = t.totls[n-1].Name
	if t.pos[2] == math.MaxUint64 {
		t.done = true
	}
	t.pos[2]++

	ret := make([]XattrTotal, n)
	for i := range ret {
		if t.totls[i].Name[0] != t.id1 || t.totls[i].Name[1] != t.id2 {
			t.done = true
			// id sequence we want is done
			return ret[:i], nil
		}
		ret[i].Count = t.totls[i].Count
		ret[i].Total = t.totls[i].Total
		ret[i].ID = t.totls[i].Name
	}
	return ret, nil
}

// Reset resets the totl query to the start of the group id again
func (t *TotalsGroup) Reset() {
	t.done = false
	t.pos[0] = t.id1
	t.pos[1] = t.id2
	t.pos[2] = 0
}

// Parent contains inode of parent and what the child inode is named within
// this parent
type Parent struct {
	Ino  uint64 // Parent inode
	Pos  uint64 // Entry directory position in parent
	Type uint8  // Entry inode type matching DT_ enum values in readdir(3)
	Ent  string // Entry name as known by parent
}

// GetParents returns all parents for the given inode
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
// If passed in buffer is nil, call will allocate its own buffer.
func GetParents(dirfd *os.File, ino uint64, b []byte) ([]Parent, error) {
	if b == nil {
		b = make([]byte, getparentBufsize)
	}

	gre := getReferringEntries{}

	gre.Entries_bytes = uint64(len(b))
	gre.Entries_ptr = uint64(uintptr(unsafe.Pointer(&b[0])))
	gre.Ino = ino

	var parents []Parent

	for {
		n, err := scoutfsctl(dirfd, IOCGETREFERRINGENTRIES, unsafe.Pointer(&gre))
		if err != nil {
			return nil, err
		}
		if n == 0 {
			break
		}

		ents, isLast, err := parseDents(b)
		if err != nil {
			return nil, err
		}

		parents = append(parents, ents...)
		if isLast {
			break
		}
	}

	return parents, nil
}

func parseDents(b []byte) ([]Parent, bool, error) {
	r := bytes.NewReader(b)
	var parents []Parent
	var isLast bool
	for {
		var err error
		var parent Parent
		parent, isLast, err = parseDent(r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, false, err
		}
		parents = append(parents, parent)
		if isLast {
			break
		}
		if r.Len() == 0 {
			break
		}
	}
	return parents, isLast, nil
}

type dirent struct {
	Dir_ino     uint64
	Dir_pos     uint64
	Ino         uint64
	Entry_bytes uint16
	Flags       uint8
	D_type      uint8
	Name_len    uint8
}

const direntSize = 29

func parseDent(r *bytes.Reader) (Parent, bool, error) {
	var dent dirent
	err := binary.Read(r, binary.LittleEndian, &dent)
	if err != nil {
		return Parent{}, false, err
	}

	b := new(strings.Builder)
	_, err = io.CopyN(b, r, int64(dent.Name_len))
	if err != nil {
		return Parent{}, false, err
	}

	pad := int(dent.Entry_bytes) - (direntSize + int(dent.Name_len))
	for i := 0; i < pad; i++ {
		_, err = r.ReadByte()
		if err != nil {
			return Parent{}, false, err
		}
	}

	return Parent{
		Ino:  dent.Dir_ino,
		Pos:  dent.Dir_pos,
		Type: dent.D_type,
		Ent:  b.String(),
	}, dent.Flags&DIRENTFLAGLAST == DIRENTFLAGLAST, nil
}

const (
	// format.h: SQ_NS_LITERAL
	quotaLiteral = 0
	// format.h: SQ_NS_PROJ
	quotaProj = 1
	// format.h: SQ_NS_UID
	quotaUID = 2
	// format.h: SQ_NS_GID
	quotaGID = 3
	// format.h: SQ_NF_SELECT
	quotaSelect = 1
	// format.h: SQ_RF_TOTL_COUNT
	quotaFlagCount = 1
)

type QuotaType uint8

func (q QuotaType) String() string {
	switch q {
	case quotaLiteral:
		return "literal"
	case quotaProj:
		return "project"
	case quotaUID:
		return "uid"
	case quotaGID:
		return "gid"
	default:
		return "unknown"
	}
}

const (
	// format.h: SQ_OP_INODE
	QuotaInode = 0
	// format.h: SQ_OP_DATA
	QuotaData = 1
)

type QuotaOp uint8

func (q QuotaOp) String() string {
	switch q {
	case QuotaInode:
		return "File"
	case QuotaData:
		return "Size"
	default:
		return "Unknown"
	}
}

type Quotas struct {
	rules []quotaRule
	iter  [2]uint64
	count int
	f     *os.File
	done  bool
}

// QuotaRule is attributes for a single quota rule
type QuotaRule struct {
	Op          QuotaOp
	QuotaValue  [3]uint64
	QuotaSource [3]uint8
	QuotaFlags  [3]uint8
	Limit       uint64
	Prioirity   uint8
	Flags       uint8
}

const (
	prioPad = 3  // length largest prio string "255"
	opPad   = -7 // length largest op string "Unknown"
)

func (q QuotaRule) String() string {
	switch q.QuotaSource[2] {
	case quotaLiteral:
		return fmt.Sprintf("P: %*v %*v Literal Limit: %v",
			prioPad, q.Prioirity, opPad, q.Op, q.Limit)
	case quotaUID:
		if q.QuotaFlags[2] == quotaSelect {
			return fmt.Sprintf("P: %*v %*v UID  [%5v] Limit: %v",
				prioPad, q.Prioirity, opPad, q.Op, q.QuotaValue[2], q.Limit)
		}
		return fmt.Sprintf("P: %*v %*v UID  general Limit: %v",
			prioPad, q.Prioirity, opPad, q.Op, q.Limit)
	case quotaGID:
		if q.QuotaFlags[2] == quotaSelect {
			return fmt.Sprintf("P: %*v %*v GID  [%5v] Limit: %v",
				prioPad, q.Prioirity, opPad, q.Op, q.QuotaValue[2], q.Limit)
		}
		return fmt.Sprintf("P: %*v %*v GID  general Limit: %v",
			prioPad, q.Prioirity, opPad, q.Op, q.Limit)
	case quotaProj:
		if q.QuotaFlags[2] == quotaSelect {
			return fmt.Sprintf("P: %*v %*v Proj [%5v] Limit: %v",
				prioPad, q.Prioirity, opPad, q.Op, q.QuotaValue[2], q.Limit)
		}
		return fmt.Sprintf("P: %*v %*v Proj general Limit: %v",
			prioPad, q.Prioirity, opPad, q.Op, q.Limit)
	}

	return q.Raw(false)
}

func (q QuotaRule) StringNoLimit() string {
	switch q.QuotaSource[2] {
	case quotaLiteral:
		return fmt.Sprintf("P: %*v %*v Literal",
			prioPad, q.Prioirity, opPad, q.Op)
	case quotaUID:
		if q.QuotaFlags[2] == quotaSelect {
			return fmt.Sprintf("P: %*v %*v UID  [%5v]",
				prioPad, q.Prioirity, opPad, q.Op, q.QuotaValue[2])
		}
		return fmt.Sprintf("P: %*v %*v UID  general",
			prioPad, q.Prioirity, opPad, q.Op)
	case quotaGID:
		if q.QuotaFlags[2] == quotaSelect {
			return fmt.Sprintf("P: %*v %*v GID  [%5v]",
				prioPad, q.Prioirity, opPad, q.Op, q.QuotaValue[2])
		}
		return fmt.Sprintf("P: %*v %*v GID  general",
			prioPad, q.Prioirity, opPad, q.Op)
	case quotaProj:
		if q.QuotaFlags[2] == quotaSelect {
			return fmt.Sprintf("P: %*v %*v Proj [%5v]",
				prioPad, q.Prioirity, opPad, q.Op, q.QuotaValue[2])
		}
		return fmt.Sprintf("P: %*v %*v Proj general",
			prioPad, q.Prioirity, opPad, q.Op)
	}

	return q.Raw(false)
}

func (q QuotaRule) HumanString() string {
	limit := fmt.Sprintf("%v", q.Limit)
	if q.Op == QuotaData {
		limit = byteToHuman(q.Limit)
	}

	switch q.QuotaSource[2] {
	case quotaLiteral:
		return fmt.Sprintf("P: %*v %*v Literal Limit: %v",
			prioPad, q.Prioirity, opPad, q.Op, limit)
	case quotaUID:
		if q.QuotaFlags[2] == quotaSelect {
			return fmt.Sprintf("P: %*v %*v UID [%5v] Limit: %v",
				prioPad, q.Prioirity, opPad, q.Op, q.QuotaValue[2], limit)
		}
		return fmt.Sprintf("P: %*v %*v UID general Limit: %v",
			prioPad, q.Prioirity, opPad, q.Op, limit)
	case quotaGID:
		if q.QuotaFlags[2] == quotaSelect {
			return fmt.Sprintf("P: %*v %*v GID [%5v] Limit: %v",
				prioPad, q.Prioirity, opPad, q.Op, q.QuotaValue[2], limit)
		}
		return fmt.Sprintf("P: %*v %*v GID general Limit: %v",
			prioPad, q.Prioirity, opPad, q.Op, limit)
	case quotaProj:
		if q.QuotaFlags[2] == quotaSelect {
			return fmt.Sprintf("P: %*v %*v Proj [%5v] Limit: %v",
				prioPad, q.Prioirity, opPad, q.Op, q.QuotaValue[2], limit)
		}
		return fmt.Sprintf("P: %*v %*v Proj general Limit: %v",
			prioPad, q.Prioirity, opPad, q.Op, limit)
	}

	return q.Raw(false)
}

func (q QuotaRule) IsGeneral() bool {
	return q.QuotaSource[2] != quotaLiteral && q.QuotaFlags[2] != quotaSelect
}

func (q QuotaRule) QuotaType() string {
	switch q.QuotaSource[2] {
	case quotaLiteral:
		return "Literal"
	case quotaUID:
		return "UID"
	case quotaGID:
		return "GID"
	case quotaProj:
		return "Proj"
	default:
		return "-"
	}
}

func (q QuotaRule) Raw(human bool) string {
	if human {
		return fmt.Sprintf("Op=%v, Value=%v, Source=%v, Flags=%v, Limit=%v Prioirty=%v",
			q.Op, q.QuotaValue, q.QuotaSource, q.QuotaFlags, byteToHuman(q.Limit), q.Prioirity)
	}
	return fmt.Sprintf("Op=%v, Value=%v, Source=%v, Flags=%v, Limit=%v Prioirty=%v",
		q.Op, q.QuotaValue, q.QuotaSource, q.QuotaFlags, q.Limit, q.Prioirity)
}

func byteToHuman(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(b)/float64(div), "KMGTPE"[exp])
}

// RuleSet is a list of quota rules, when sorted these
// will be in the order as the filesystem would match them
type RuleSet []QuotaRule

// Len returns the length of the ruleset
func (r RuleSet) Len() int { return len(r) }

// Less returns true if the i-th rule would be matched before the j-th rule
func (r RuleSet) Less(i, j int) bool {
	if r[i].Prioirity != r[j].Prioirity {
		// higher priority is matched first
		return r[i].Prioirity > r[j].Prioirity
	}

	if r[i].QuotaValue[0] != r[j].QuotaValue[0] {
		return r[i].QuotaValue[0] > r[j].QuotaValue[0]
	}
	if r[i].QuotaSource[0] != r[j].QuotaSource[0] {
		return r[i].QuotaSource[0] > r[j].QuotaSource[0]
	}
	if r[i].QuotaFlags[0] != r[j].QuotaFlags[0] {
		return r[i].QuotaFlags[0] > r[j].QuotaFlags[0]
	}

	if r[i].QuotaValue[1] != r[j].QuotaValue[1] {
		return r[i].QuotaValue[1] > r[j].QuotaValue[1]
	}
	if r[i].QuotaSource[1] != r[j].QuotaSource[1] {
		return r[i].QuotaSource[1] > r[j].QuotaSource[1]
	}
	if r[i].QuotaFlags[1] != r[j].QuotaFlags[1] {
		return r[i].QuotaFlags[1] > r[j].QuotaFlags[1]
	}

	if r[i].QuotaValue[2] != r[j].QuotaValue[2] {
		return r[i].QuotaValue[2] > r[j].QuotaValue[2]
	}
	if r[i].QuotaSource[2] != r[j].QuotaSource[2] {
		return r[i].QuotaSource[2] > r[j].QuotaSource[2]
	}
	if r[i].QuotaFlags[2] != r[j].QuotaFlags[2] {
		return r[i].QuotaFlags[2] > r[j].QuotaFlags[2]
	}

	if r[i].Op != r[j].Op {
		return r[i].Op > r[j].Op
	}

	if r[i].Limit != r[j].Limit {
		return r[i].Limit > r[j].Limit
	}

	// rules are the same (should never happen)
	return false
}

// Swap will swap the i-th and j-th elements in the ruleset
func (r RuleSet) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

// GetQuotaRules initalizes reading the current quota set.
// Quota rules are not returned in sorted order, so to get
// the order which they are matched the full list must be
// collected then sorted.
func GetQuotaRules(f *os.File, count int) (*Quotas, error) {
	if count < 1 {
		return nil, fmt.Errorf("must provide count > 0")
	}
	rules := make([]quotaRule, count)

	return &Quotas{rules: rules, f: f, count: count}, nil
}

// Next returns next batch of quota rules.
func (q *Quotas) Next() ([]QuotaRule, error) {
	if q.done {
		return nil, nil
	}

	query := getQuotaRules{
		Iterator: q.iter,
		Ptr:      uint64(uintptr(unsafe.Pointer(&q.rules[0]))),
		Nr:       uint64(q.count),
	}

	n, err := scoutfsctl(q.f, IOCGETQUOTARULES, unsafe.Pointer(&query))
	if err != nil {
		return nil, err
	}
	if n == 0 {
		q.done = true
		return nil, nil
	}

	ret := make([]QuotaRule, n)
	for i := range ret {
		ret[i].Op = QuotaOp(q.rules[i].Op)
		ret[i].QuotaValue = q.rules[i].Name_val
		ret[i].QuotaFlags = q.rules[i].Name_flags
		ret[i].QuotaSource = q.rules[i].Name_source
		ret[i].Limit = q.rules[i].Limit
		ret[i].Prioirity = q.rules[i].Prio
		ret[i].Flags = q.rules[i].Rule_flags
	}

	q.iter = query.Iterator

	return ret, nil
}

// Reset resets the quota listing to the start
func (t *Quotas) Reset() {
	t.done = false
	t.iter = [2]uint64{}
}

func QuotaDelete(f *os.File, q QuotaRule) error {
	qr := quotaRule{
		Name_val:    q.QuotaValue,
		Limit:       q.Limit,
		Prio:        q.Prioirity,
		Op:          uint8(q.Op),
		Rule_flags:  q.Flags,
		Name_source: q.QuotaSource,
		Name_flags:  q.QuotaFlags,
	}

	_, err := scoutfsctl(f, IOCDELQUOTARULE, unsafe.Pointer(&qr))
	return err
}

func QuotaAddDataLiteral(f *os.File, id1, id2, id3, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaData,
		QuotaValue:  [3]uint64{id1, id2, id3},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaLiteral},
		Limit:       limit,
		Prioirity:   priority,
	})
}

func QuotaAddInodeLiteral(f *os.File, id1, id2, id3, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaInode,
		QuotaValue:  [3]uint64{id1, id2, id3},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaLiteral},
		Limit:       limit,
		Prioirity:   priority,
		Flags:       quotaFlagCount,
	})
}

func QuotaAddDataProjectGeneral(f *os.File, id1, id2, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaData,
		QuotaValue:  [3]uint64{id1, id2, 0},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaProj},
		Limit:       limit,
		Prioirity:   priority,
	})
}

func QuotaAddInodeProjectGeneral(f *os.File, id1, id2, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaInode,
		QuotaValue:  [3]uint64{id1, id2, 0},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaProj},
		Limit:       limit,
		Prioirity:   priority,
		Flags:       quotaFlagCount,
	})
}

func QuotaAddDataProject(f *os.File, id1, id2, project, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaData,
		QuotaValue:  [3]uint64{id1, id2, project},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaProj},
		QuotaFlags:  [3]uint8{0, 0, quotaSelect},
		Limit:       limit,
		Prioirity:   priority,
	})
}

func QuotaAddInodeProject(f *os.File, id1, id2, project, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaInode,
		QuotaValue:  [3]uint64{id1, id2, project},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaProj},
		QuotaFlags:  [3]uint8{0, 0, quotaSelect},
		Limit:       limit,
		Prioirity:   priority,
		Flags:       quotaFlagCount,
	})
}

func QuotaAddDataUIDGeneral(f *os.File, id1, id2, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaData,
		QuotaValue:  [3]uint64{id1, id2, 0},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaUID},
		Limit:       limit,
		Prioirity:   priority,
	})
}

func QuotaAddInodeUIDGeneral(f *os.File, id1, id2, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaInode,
		QuotaValue:  [3]uint64{id1, id2, 0},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaUID},
		Limit:       limit,
		Prioirity:   priority,
		Flags:       quotaFlagCount,
	})
}

func QuotaAddDataUID(f *os.File, id1, id2, uid, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaData,
		QuotaValue:  [3]uint64{id1, id2, uid},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaUID},
		QuotaFlags:  [3]uint8{0, 0, quotaSelect},
		Limit:       limit,
		Prioirity:   priority,
	})
}

func QuotaAddInodeUID(f *os.File, id1, id2, uid, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaInode,
		QuotaValue:  [3]uint64{id1, id2, uid},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaUID},
		QuotaFlags:  [3]uint8{0, 0, quotaSelect},
		Limit:       limit,
		Prioirity:   priority,
		Flags:       quotaFlagCount,
	})
}

func QuotaAddDataGIDGeneral(f *os.File, id1, id2, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaData,
		QuotaValue:  [3]uint64{id1, id2, 0},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaGID},
		Limit:       limit,
		Prioirity:   priority,
	})
}

func QuotaAddInodeGIDGeneral(f *os.File, id1, id2, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaInode,
		QuotaValue:  [3]uint64{id1, id2, 0},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaGID},
		Limit:       limit,
		Prioirity:   priority,
		Flags:       quotaFlagCount,
	})
}

func QuotaAddDataGID(f *os.File, id1, id2, gid, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaData,
		QuotaValue:  [3]uint64{id1, id2, gid},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaGID},
		QuotaFlags:  [3]uint8{0, 0, quotaSelect},
		Limit:       limit,
		Prioirity:   priority,
	})
}

func QuotaAddInodeGID(f *os.File, id1, id2, gid, limit uint64, priority uint8) error {
	return quotaAdd(f, QuotaRule{
		Op:          QuotaInode,
		QuotaValue:  [3]uint64{id1, id2, gid},
		QuotaSource: [3]uint8{quotaLiteral, quotaLiteral, quotaGID},
		QuotaFlags:  [3]uint8{0, 0, quotaSelect},
		Limit:       limit,
		Prioirity:   priority,
		Flags:       quotaFlagCount,
	})
}

func quotaAdd(f *os.File, q QuotaRule) error {
	qr := quotaRule{
		Name_val:    q.QuotaValue,
		Limit:       q.Limit,
		Prio:        q.Prioirity,
		Op:          uint8(q.Op),
		Rule_flags:  q.Flags,
		Name_source: q.QuotaSource,
		Name_flags:  q.QuotaFlags,
	}

	_, err := scoutfsctl(f, IOCADDQUOTARULE, unsafe.Pointer(&qr))
	return err
}

func GetProjectID(f *os.File) (uint64, error) {
	var projectid uint64
	_, err := scoutfsctl(f, IOCGETPROJECTID, unsafe.Pointer(&projectid))
	return projectid, err
}

func SetProjectID(f *os.File, projectid uint64) error {
	_, err := scoutfsctl(f, IOCSETPROJECTID, unsafe.Pointer(&projectid))
	return err
}
