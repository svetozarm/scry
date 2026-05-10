package update

import "os"

// writable checks if dir is writable by creating and removing a temp file.
func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".scry-write-check-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}
