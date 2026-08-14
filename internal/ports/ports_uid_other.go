//go:build !linux

package ports

import "io/fs"

// owners is never called outside Linux, but retaining a portable implementation
// keeps the command package buildable for release archives on every platform.
func ownerUID(fs.FileInfo) int { return 0 }
