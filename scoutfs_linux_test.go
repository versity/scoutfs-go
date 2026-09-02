package scoutfs

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"
)

func TestInodePathsOnScoutFS(t *testing.T) {
	root := os.Getenv("SCOUTFS_TEST_MOUNT")
	if root == "" {
		t.Skip("set SCOUTFS_TEST_MOUNT to a writable ScoutFS mount root")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	dirfd, err := os.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer dirfd.Close()
	dir, err := os.MkdirTemp(root, "inode-path-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	name := filepath.Join(dir, strings.Repeat("a", 200))
	if err := os.WriteFile(name, []byte("path lookup"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "second-link")
	if err := os.Link(name, link); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	ino := fi.Sys().(*syscall.Stat_t).Ino
	want := []string{filepath.Join(filepath.Base(dir), filepath.Base(name)), filepath.Join(filepath.Base(dir), "second-link")}
	sort.Strings(want)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
			}
		}
	}()
	defer func() { close(stop); <-done }()
	for i := 0; i < 256; i++ {
		path, err := InoToPath(dirfd, ino)
		if err != nil || (path != want[0] && path != want[1]) {
			t.Fatalf("InoToPath = %q, %v; want one of %v", path, err, want)
		}
		paths, err := InoToPaths(dirfd, ino)
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(paths)
		if !reflect.DeepEqual(paths, want) {
			t.Fatalf("InoToPaths = %v, want %v", paths, want)
		}
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if path, err := InoToPath(dirfd, ino); path != "" || !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("deleted InoToPath = %q, %v; want empty path and ENOENT", path, err)
	}
	if paths, err := InoToPaths(dirfd, ino); len(paths) != 0 || err != nil {
		t.Fatalf("deleted InoToPaths = %v, %v; want no paths and no error", paths, err)
	}
}
