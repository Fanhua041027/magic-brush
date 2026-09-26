//go:build !windows

package config

import "os"

func atomicReplaceFile(src, dst string) error {
	return os.Rename(src, dst)
}
