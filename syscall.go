package scoutfs

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	// SYS_OPENBYHANDLEAT linux system call
	SYS_OPENBYHANDLEAT = 304
	// FILEID_SCOUTFS for scoutfs file handle
	FILEID_SCOUTFS = 0x81
)

// OpenByHandle is similar to OpenByID, but returns just the file descriptor
// and does not have the added overhead of getting the filename
// An open file within scoutfs is supplied for ioctls
// (usually just the base mount point directory)
func OpenByHandle(dirfd *os.File, ino uint64, flags int) (uintptr, error) {
	h := &FileHandle{
		FidSize:    uint32(unsafe.Sizeof(FileID{})),
		HandleType: FILEID_SCOUTFS,
		FID:        FileID{Ino: ino},
	}
	return openbyhandleat(dirfd.Fd(), h, flags)
}

func openbyhandleat(dirfd uintptr, handle *FileHandle, flags int) (uintptr, error) {
	fd, _, e1 := syscall.Syscall6(SYS_OPENBYHANDLEAT, dirfd, uintptr(unsafe.Pointer(handle)), uintptr(flags), 0, 0, 0)
	var err error
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return fd, err
}

func scoutfsctl(fd, cmd, ptr uintptr) (int, error) {
	count, _, e1 := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
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