// Copyright (c) 2018 Versity Software, Inc.
//
// Use of this source code is governed by a BSD-3-Clause license
// that can be found in the LICENSE file in the root of the source
// tree.

package scoutfs

import (
	"bytes"
	"encoding/binary"
	"os"
	"syscall"
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
func NewQuery(path string, opts ...Option) (*Query, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	q := &Query{
		//default batch size is 128
		batch: 128,
		fsfd:  f,
	}

	for _, opt := range opts {
		opt(q)
	}

	return q, nil
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
	q.first.Minor++
	if q.first.Ino == 0 {
		q.first.Minor++
		if q.first.Minor == 0 {
			q.first.Major++
		}
	}

	return inodes, nil
}

// Close queryHandle and cleanup
func (q *Query) Close() {
	q.fsfd.Close()
}

// StatMore returns scoutfs specific metadata for path
func StatMore(path string) (Stat, error) {
	f, err := os.Open(path)
	if err != nil {
		return Stat{}, err
	}
	defer f.Close()

	var s Stat

	_, err = scoutfsctl(f.Fd(), IOCSTATMORE, uintptr(unsafe.Pointer(&s)))
	if err != nil {
		return Stat{}, err
	}

	return s, nil
}

func scoutfsctl(fd, cmd, ptr uintptr) (int, error) {
	count, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
	if err != 0 {
		return 0, err
	}
	return int(count), nil
}
