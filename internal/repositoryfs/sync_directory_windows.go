//go:build windows

package repositoryfs

// SyncDirectory is a no-op because Windows does not support syncing an opened
// directory handle. The temporary file is still flushed before replacement.
func SyncDirectory(_ string) error {
	return nil
}
