package scoutfs

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	// sysOpenByHandleAt linux system call
	sysOpenByHandleAt = 304
	// fileIDScoutfs for scoutfs file handle
	fileIDScoutfs = 0x81
)

type fileID struct {
	Ino       uint64
	ParentIno uint64
}

type fileHandle struct {
	FidSize    uint32
	HandleType int32
	FID        fileID
}

// OpenByHandle is similar to OpenByID, but returns just the file descriptor
// and does not have the added overhead of getting the filename
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func OpenByHandle(dirfd *os.File, ino uint64, flags int) (uintptr, error) {
	h := &fileHandle{
		FidSize:    uint32(unsafe.Sizeof(fileID{})),
		HandleType: fileIDScoutfs,
		FID:        fileID{Ino: ino},
	}
	return openbyhandleat(dirfd, h, flags)
}

func openbyhandleat(dirfd *os.File, handle *fileHandle, flags int) (uintptr, error) {
	fd, _, e1 := syscall.Syscall6(sysOpenByHandleAt, uintptr(dirfd.Fd()), uintptr(unsafe.Pointer(handle)), uintptr(flags), 0, 0, 0)
	var err error
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return fd, err
}

func scoutfsctl(f *os.File, cmd int, ptr unsafe.Pointer) (int, error) {
	count, _, e1 := syscall.Syscall(syscall.SYS_IOCTL, uintptr(f.Fd()), uintptr(cmd), uintptr(ptr))
	var err error
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return int(count), err
}

// Do the interface allocations only once for common
// Errno values.
var (
	errEAGAIN error = syscall.EAGAIN
	errEINVAL error = syscall.EINVAL
	errENOENT error = syscall.ENOENT
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case syscall.EAGAIN:
		return errEAGAIN
	case syscall.EINVAL:
		return errEINVAL
	case syscall.ENOENT:
		return errENOENT
	}
	return e
}
