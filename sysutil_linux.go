package taki

import (
	"fmt"
	"syscall"
)

func GetInode(path string) {
	var info os.FileInfo
	if info, err = os.Stat(path); err != nil {
		return
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Ino, nil
	} else {
		return 0, fmt.Errorf("failed to stat file '%s'", path)
	}
}
