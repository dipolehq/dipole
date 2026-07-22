//go:build linux

/*
process.go implements process identification on Linux by mapping a TCP source
port to a PID via /proc/net/tcp and then reading process metadata from /proc.
Only active when DIPOLE_CLIENT_WORKLOAD_PROCESS_IDENTIFICATION_ENABLED=true.

/proc/net/tcp format: fields are whitespace-separated, field[1] is local_address
as "HexIP:HexPort", field[9] is the socket inode. /proc/<pid>/cmdline uses null
bytes as separators. /proc/<pid>/exe may have " (deleted)" suffix if the binary
was replaced. UID resolution falls back to the numeric UID if /etc/passwd lookup fails.
*/
package identity

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

func gatherProcess(sourcePort int) *pb.ProcessEvidence {
	if sourcePort <= 0 {
		return nil
	}

	inode, err := findInodeByPort(sourcePort)
	if err != nil || inode == "" {
		return nil
	}

	pid, err := findPIDByInode(inode)
	if err != nil || pid == "" {
		return nil
	}

	procPath := readLink(fmt.Sprintf("/proc/%s/exe", pid))
	cmdLine := readCmdline(fmt.Sprintf("/proc/%s/cmdline", pid))
	name, userName := readStatus(fmt.Sprintf("/proc/%s/status", pid))

	if procPath == "" && name == "" {
		return nil
	}

	return &pb.ProcessEvidence{
		Name:        name,
		Path:        procPath,
		CommandLine: cmdLine,
		UserName:    userName,
	}
}

func findInodeByPort(port int) (string, error) {
	f, err := os.Open("/proc/net/tcp")
	if err != nil {
		return "", err
	}
	defer f.Close()

	portHex := strings.ToUpper(fmt.Sprintf("%X", port))

	scanner := bufio.NewScanner(f)
	scanner.Scan()
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		addrParts := strings.Split(fields[1], ":")
		if len(addrParts) != 2 {
			continue
		}
		if strings.ToUpper(addrParts[1]) == portHex {
			return fields[9], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("port %d not found in /proc/net/tcp", port)
}

func findPIDByInode(inode string) (string, error) {
	target := "socket:[" + inode + "]"

	procDirs, err := os.ReadDir("/proc")
	if err != nil {
		return "", err
	}

	for _, d := range procDirs {
		if !d.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(d.Name()); err != nil {
			continue
		}

		fdDir := filepath.Join("/proc", d.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}
			if link == target {
				return d.Name(), nil
			}
		}
	}

	return "", fmt.Errorf("inode %s not found in any process", inode)
}

func readLink(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	target = strings.TrimSuffix(target, " (deleted)")
	return target
}

func readCmdline(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	return strings.Join(parts, " ")
}

func readStatus(path string) (name, userName string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		}
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(strings.TrimPrefix(line, "Uid:"))
			if len(fields) > 0 {
				userName = resolveUID(fields[0])
			}
		}
	}
	_ = scanner.Err()
	return name, userName
}

func resolveUID(uid string) string {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return uid
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) >= 3 && parts[2] == uid {
			return parts[0]
		}
	}
	_ = scanner.Err()
	return uid
}
