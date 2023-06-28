// -build linux

package tkserver

import "errors"

func GetInode(path string) (uint, error) {
	return 0, errors.New("cannot get inode on windows")
}
