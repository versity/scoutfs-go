package scoutfs

import (
	"errors"
	"os"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestInoPathLayout(t *testing.T) {
	var request inoPathRequest
	var abi inoPath
	var result inoPathResult
	for name, layout := range map[string]struct{ got, want uintptr }{
		"request size":   {unsafe.Sizeof(request), 40},
		"generated size": {unsafe.Sizeof(request), unsafe.Sizeof(abi)},
		"inode":          {unsafe.Offsetof(request.Ino), unsafe.Offsetof(abi.Ino)},
		"directory":      {unsafe.Offsetof(request.Dir_ino), unsafe.Offsetof(abi.Dir_ino)},
		"position":       {unsafe.Offsetof(request.Dir_pos), unsafe.Offsetof(abi.Dir_pos)},
		"pointer ABI":    {unsafe.Offsetof(request.Result_ptr), unsafe.Offsetof(abi.Result_ptr)},
		"length ABI":     {unsafe.Offsetof(request.Result_bytes), unsafe.Offsetof(abi.Result_bytes)},
		"result pointer": {unsafe.Offsetof(request.Result_ptr), 24},
		"result bytes":   {unsafe.Offsetof(request.Result_bytes), 32},
		"path size":      {unsafe.Offsetof(result.PathSize), 16},
		"path data":      {unsafe.Offsetof(result.Path), 24},
		"result size":    {unsafe.Sizeof(result), 24 + pathmax},
	} {
		if layout.got != layout.want {
			t.Errorf("%s = %d, want %d", name, layout.got, layout.want)
		}
	}
}

func TestInoToPathErrors(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		name, err := InoToPath(f, 11293)
		if !errors.Is(err, syscall.EBADF) || name != "" {
			t.Fatalf("InoToPath = %q, %v; want empty path and EBADF", name, err)
		}
		names, err := InoToPaths(f, 11293)
		if !errors.Is(err, syscall.EBADF) || names != nil {
			t.Fatalf("InoToPaths = %v, %v; want nil paths and EBADF", names, err)
		}
	}
}

//go:noinline
func growInodePathTestStack(depth int) byte {
	var pad [1024]byte
	pad[depth%len(pad)] = byte(depth)
	if depth > 0 {
		pad[0] = growInodePathTestStack(depth - 1)
	}
	runtime.KeepAlive(&pad)
	return pad[depth%len(pad)]
}

//go:noinline
func fillInodePathTestResult(request *inoPathRequest) {
	growInodePathTestStack(64)
	request.Result_ptr.PathSize = 7
	copy(request.Result_ptr.Path[:], "test/f\x00")
}

func TestInoPathResultSurvivesStackGrowth(t *testing.T) {
	done := make(chan bool)
	go func() {
		var res inoPathResult
		request := inoPathRequest{Ino: 11293, Result_ptr: &res, Result_bytes: uint16(unsafe.Sizeof(res))}
		fillInodePathTestResult(&request)
		done <- request.Result_ptr == &res && res.PathSize == 7 && string(res.Path[:7]) == "test/f\x00"
	}()
	if !<-done {
		t.Fatal("request result pointer did not follow its buffer across stack growth")
	}
}
