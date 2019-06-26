// Copyright (c) 2018 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

package scoutfs

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"time"
	"unsafe"
)

const (
	max64 = 0xffffffffffffffff
	max32 = 0xffffffff
)

// Query to keep track of in-process query
type Query struct {
	first InodesEntry
	last  InodesEntry
	index uint8
	batch uint32
	fsfd  *os.File
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
	buf := make([]byte, int(unsafe.Sizeof(InodesEntry{}))*int(q.batch))
	query := queryInodes{
		first:   q.first,
		last:    q.last,
		entries: uintptr(unsafe.Pointer(&buf[0])),
		count:   q.batch,
		index:   q.index,
	}

	n, err := scoutfsctl(q.fsfd.Fd(), IOCQUERYINODES, uintptr(unsafe.Pointer(&query)))
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	rbuf := bytes.NewReader(buf)
	var inodes []InodesEntry

	var e InodesEntry
	for i := 0; i < n; i++ {
		err := binary.Read(rbuf, binary.LittleEndian, &e)
		if err != nil {
			return nil, err
		}

		inodes = append(inodes, e)
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
	s := Stat{ValidBytes: uint64(unsafe.Sizeof(Stat{}))}

	_, err := scoutfsctl(f.Fd(), IOCSTATMORE, uintptr(unsafe.Pointer(&s)))
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
		dataVersion: version,
		iSize:       size,
		flags:       flags,
		ctimesec:    uint64(ctime.Unix()),
		ctimensec:   uint32(nsec),
	}

	_, err := scoutfsctl(f.Fd(), IOCSETATTRMORE, uintptr(unsafe.Pointer(&s)))
	return err
}

// InoToPath converts an inode number to a path in the filesystem
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func InoToPath(dirfd *os.File, ino uint64) (string, error) {
	var res InoPathResult
	ip := InoPath{
		Ino:        ino,
		ResultPtr:  uint64(uintptr(unsafe.Pointer(&res))),
		ResultSize: uint16(unsafe.Sizeof(res)),
	}

	_, err := scoutfsctl(dirfd.Fd(), IOCINOPATH, uintptr(unsafe.Pointer(&ip)))
	if err != nil {
		return "", err
	}

	b := bytes.Trim(res.Path[:res.PathSize], "\x00")

	return string(b), nil
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
	r := IocRelease{
		Count:       math.MaxUint64,
		DataVersion: version,
	}

	_, err := scoutfsctl(f.Fd(), IOCRELEASE, uintptr(unsafe.Pointer(&r)))
	return err
}

// StageFile rehydrates offline file
func StageFile(path string, version, offset uint64, b []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	return FStageFile(f, version, offset, b)
}

// FStageFile rehydrates offline file
func FStageFile(f *os.File, version, offset uint64, b []byte) error {
	r := IocStage{
		DataVersion: version,
		BufPtr:      uint64(uintptr(unsafe.Pointer(&b[0]))),
		Offset:      offset,
		count:       int32(len(b)),
	}

	_, err := scoutfsctl(f.Fd(), IOCSTAGE, uintptr(unsafe.Pointer(&r)))
	return err
}

// Waiters to keep track of data waiters
type Waiters struct {
	ino    uint64
	iblock uint64
	batch  uint16
	fsfd   *os.File
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
	buf := make([]byte, int(unsafe.Sizeof(DataWaitingEntry{}))*int(w.batch))
	dataWaiting := dataWaiting{
		afterIno:    w.ino,
		afterIblock: w.iblock,
		entries:     uintptr(unsafe.Pointer(&buf[0])),
		count:       w.batch,
	}

	n, err := scoutfsctl(w.fsfd.Fd(), IOCDATAWAITING, uintptr(unsafe.Pointer(&dataWaiting)))
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	rbuf := bytes.NewReader(buf)
	var inodes []DataWaitingEntry

	var e DataWaitingEntry
	for i := 0; i < n; i++ {
		err := binary.Read(rbuf, binary.LittleEndian, &e.Ino)
		if err != nil {
			return nil, err
		}
		err = binary.Read(rbuf, binary.LittleEndian, &e.Iblock)
		if err != nil {
			return nil, err
		}
		err = binary.Read(rbuf, binary.LittleEndian, &e.Op)
		if err != nil {
			return nil, err
		}

		inodes = append(inodes, e)
	}

	w.ino = inodes[n-1].Ino
	w.iblock = inodes[n-1].Iblock

	return inodes, nil
}

// XattrQuery to keep track of in-process xattr query
type XattrQuery struct {
	next  uint64
	batch uint32
	key   string
	fsfd  *os.File
}

// NewXattrQuery creates a new scoutfs Xattr Query
// Specify query xattr key
// and specify optinally batching with WithXBatchSize()
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func NewXattrQuery(f *os.File, key string, opts ...XOption) *XattrQuery {
	q := &XattrQuery{
		//default batch size is 128
		batch: 128,
		key:   key,
		fsfd:  f,
	}

	for _, opt := range opts {
		opt(q)
	}

	return q
}

// XOption sets various options for NewXattrQuery
type XOption func(*XattrQuery)

// WithXBatchSize sets the max number of inodes to be returned at a time
func WithXBatchSize(size uint32) XOption {
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
	buf := make([]byte, 8*int(q.batch))
	name := []byte(q.key)
	query := findXattrs{
		nextIno:    q.next,
		name:       uintptr(unsafe.Pointer(&name[0])),
		inodesBuf:  uintptr(unsafe.Pointer(&buf[0])),
		nameSize:   uint16(len(name)),
		inodeCount: uint16(q.batch),
	}

	n, err := scoutfsctl(q.fsfd.Fd(), IOCFINDXATTRS, uintptr(unsafe.Pointer(&query)))
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	rbuf := bytes.NewReader(buf)
	var inodes []uint64

	var e uint64
	for i := 0; i < n; i++ {
		err := binary.Read(rbuf, binary.LittleEndian, &e)
		if err != nil {
			return nil, err
		}

		inodes = append(inodes, e)
	}

	q.next = e
	q.next++

	return inodes, nil
}

// ListXattrRaw holds info for iterating on xattrs
type ListXattrRaw struct {
	lxr *listXattrRaw
	f   *os.File
}

// NewListXattrRaw will list all scoutfs xattrs (including hidden) for file
func NewListXattrRaw(f *os.File) *ListXattrRaw {
	return &ListXattrRaw{
		f:   f,
		lxr: &listXattrRaw{},
	}
}

// Next gets next set of results, complete when string slice is nil
func (l *ListXattrRaw) Next() ([]string, error) {
	l.lxr.bufSize = 256 * 1024
	buf := make([]byte, 256*1024)
	l.lxr.buf = uintptr(unsafe.Pointer(&buf[0]))

	n, err := scoutfsctl(l.f.Fd(), IOCFINDXATTRS, uintptr(unsafe.Pointer(&l.lxr)))
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	return bufToStrings(buf[:n]), nil
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
