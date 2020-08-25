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
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	max64           = 0xffffffffffffffff
	max32           = 0xffffffff
	pathmax         = 4096
	sysscoutfs      = "/sys/fs/scoutfs/"
	leaderfile      = "quorum/is_leader"
	serveraddr      = "mount_options/server_addr"
	listattrBufsize = 256 * 1024
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

	q.first = e
	q.first.Ino++
	if q.first.Ino == 0 {
		q.first.Minor++
		if q.first.Minor == 0 {
			q.first.Major++
		}
	}

	return inodes, nil
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
	s := Stat{Valid_bytes: uint64(unsafe.Sizeof(Stat{}))}

	_, err := scoutfsctl(f, IOCSTATMORE, unsafe.Pointer(&s))
	if err != nil {
		return Stat{}, err
	}

	return s, nil
}

// SetAttrMore sets special scoutfs attributes
func SetAttrMore(path string, version, size, flags uint64, ctime time.Time) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return FSetAttrMore(f, version, size, flags, ctime)
}

// FSetAttrMore sets special scoutfs attributes for file handle
func FSetAttrMore(f *os.File, version, size, flags uint64, ctime time.Time) error {
	var nsec int32
	if ctime.UnixNano() == int64(int32(ctime.UnixNano())) {
		nsec = int32(ctime.UnixNano())
	}
	s := setattrMore{
		Data_version: version,
		I_size:       size,
		Flags:        flags,
		Ctime_sec:    uint64(ctime.Unix()),
		Ctime_nsec:   uint32(nsec),
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

// ReleaseFile marks file offline and frees associated extents
func ReleaseFile(path string, version uint64) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	return FReleaseFile(f, version)
}

// FReleaseFile marks file offline and frees associated extents
func FReleaseFile(f *os.File, version uint64) error {
	r := iocRelease{
		Count:   math.MaxUint64,
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
		Count:        int32(len(b)),
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
	stfs := statfsMore{Valid_bytes: sizeofstatfsMore}

	_, err := scoutfsctl(f, IOCSTATFSMORE, unsafe.Pointer(&stfs))
	if err != nil {
		return FSID{}, err
	}
	if stfs.Valid_bytes != sizeofstatfsMore {
		return FSID{}, fmt.Errorf("unexpected return size: %v", stfs.Valid_bytes)
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

// IsLeader returns true if the path is the current scoutfs leader mount
func IsLeader(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open: %v", err)
	}
	defer f.Close()

	id, err := GetIDs(f)
	if err != nil {
		return false, fmt.Errorf("error GetIDs: %v", err)
	}

	sfspath := filepath.Join(sysscoutfs, id.ShortID, leaderfile)
	b, err := ioutil.ReadFile(sfspath)
	if err != nil {
		return false, fmt.Errorf("read %q: %v", sfspath, err)
	}

	return strings.TrimSuffix(string(b), "\n") == "1", nil
}

// GetServerAddr returns the server IP address string.  This is only valid
// if run on a quorum member mount.
func GetServerAddr(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open: %v", err)
	}
	defer f.Close()

	id, err := GetIDs(f)
	if err != nil {
		return "", fmt.Errorf("error GetIDs: %v", err)
	}

	sfspath := filepath.Join(sysscoutfs, id.ShortID, serveraddr)
	b, err := ioutil.ReadFile(sfspath)
	if err != nil {
		return "", fmt.Errorf("read %q: %v", sfspath, err)
	}

	fields := strings.Split(string(b), ":")
	if len(fields) != 2 {
		return "", fmt.Errorf("unknown addr")
	}

	return fields[0], nil
}
