package utils

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"k8s.io/klog/v2"
)

func CreateTempFile(contents string) (filename string, reterr error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	defer func() {
		if err = f.Close(); err != nil {
			return
		}
	}()
	_, err = f.WriteString(contents)
	if err != nil {
		return "", err
	}
	return f.Name(), nil
}

func CreateTempDir() (dir string, err error) {
	return os.MkdirTemp("", "")
}

func DetermineHostByteOrder() binary.ByteOrder {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	if b == 0x04 {
		return binary.LittleEndian
	}

	return binary.BigEndian
}

const (
	SystemProcessName      string = "system_processes"
	SystemProcessNamespace string = "system"
)

func GetPathFromPID(searchPath string, pid uint64) (string, error) {
	path := fmt.Sprintf(searchPath, pid)
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open cgroup description file for pid %d: %v", pid, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "pod") || strings.Contains(line, "containerd") || strings.Contains(line, "crio") {
			return line, nil
		}
		break
	}
	return "", fmt.Errorf("could not find cgroup description entry for pid %d", pid)
}

func GetPIDNamespace(pid uint64) uint64 {
	link := fmt.Sprintf("/proc/%d/ns/pid", pid)

	fd, err := os.Open(link)
	if err != nil {
		klog.V(1).Infof("cannot open %s: %v", link, err)
		return 0
	}
	defer fd.Close()

	stat, err := fd.Stat()
	if err != nil {
		klog.V(1).Infof("cannot stat %s: %v", link, err)
		return 0
	}

	return stat.Sys().(*syscall.Stat_t).Ino
}
