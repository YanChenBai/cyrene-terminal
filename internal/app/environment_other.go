//go:build !windows

package app

import "os"

func refreshedSearchPath() string { return os.Getenv("PATH") }
