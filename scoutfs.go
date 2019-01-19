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

// NewQuery creates a new scoutfs WalkHandle
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

// Option sets various options for NewWalkHandle
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

	pack, err := query.pack()
	if err != nil {
		return []InodesEntry{}, err
	}

	n, err := scoutfsctl(q.fsfd.Fd(), IOCQUERYINODES, uintptr(unsafe.Pointer(&pack)))
	if err != nil {
		return []InodesEntry{}, err
	}

	if n == 0 {
		return []InodesEntry{}, nil
	}

	rbuf := bytes.NewReader(buf)
	var inodes []InodesEntry

	var e InodesEntry
	for i := 0; i < n; i++ {
		//packed scoutfs_ioctl_walk_inodes_entry requires
		//unpacking each member individually
		err := binary.Read(rbuf, binary.LittleEndian, &e.Major)
		if err != nil {
			return []InodesEntry{}, err
		}
		err = binary.Read(rbuf, binary.LittleEndian, &e.Minor)
		if err != nil {
			return []InodesEntry{}, err
		}
		err = binary.Read(rbuf, binary.LittleEndian, &e.Ino)
		if err != nil {
			return []InodesEntry{}, err
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
