// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package node

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/tsdb/fileutil"
)

// legacyInstanceNames lists historical instance directory names under DataDir.
// Order matters: newer names are tried first.
var legacyInstanceNames = []string{"tomo", "geth"}

// migrateLegacyInstanceDir renames a legacy instance directory (tomo, geth) to
// the current instance name (e.g. viction) when the target does not exist yet.
func migrateLegacyInstanceDir(dataDir, targetName string) (fileutil.Releaser, error) {
	if dataDir == "" || targetName == "" {
		return nil, nil
	}
	target := filepath.Join(dataDir, targetName)
	if instanceDirHasData(target) {
		return nil, nil
	}
	for _, legacy := range legacyInstanceNames {
		if legacy == targetName {
			continue
		}
		src := filepath.Join(dataDir, legacy)
		if !instanceDirHasData(src) {
			continue
		}
		if common.FileExist(target) {
			return nil, fmt.Errorf("cannot migrate instance datadir from %q to %q: destination path already exists; remove or rename %q, then restart", src, target, target)
		}
		release, _, err := fileutil.Flock(filepath.Join(src, "LOCK"))
		if err != nil {
			return nil, err
		}

		log.Info("Migrating instance datadir", "from", src, "to", target)
		if err := os.Rename(src, target); err != nil {
			return nil, err
		}
		return release, nil
	}
	return nil, nil
}

func instanceDirHasData(dir string) bool {
	for _, name := range []string{"chaindata", "LOCK", datadirPrivateKey} {
		if common.FileExist(filepath.Join(dir, name)) {
			return true
		}
	}
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) > 0
}
