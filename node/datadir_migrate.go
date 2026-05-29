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

// prepareInstanceDirectory resolves {dataDir}/{targetName} for the running node.
// When a legacy directory (tomo, geth) must be renamed to the target name, the
// legacy LOCK is acquired before os.Rename so a concurrent node still using the
// legacy path prevents the rename (and returns ErrDatadirUsed).
func prepareInstanceDirectory(dataDir, targetName string) (instdir string, release fileutil.Releaser, err error) {
	if dataDir == "" || targetName == "" {
		return "", nil, nil
	}
	target := filepath.Join(dataDir, targetName)

	if !instanceDirHasData(target) {
		for _, legacy := range legacyInstanceNames {
			if legacy == targetName {
				continue
			}
			src := filepath.Join(dataDir, legacy)
			if !instanceDirHasData(src) {
				continue
			}
			if common.FileExist(target) {
				return "", nil, fmt.Errorf("cannot migrate instance datadir from %q to %q: destination path already exists; remove or rename %q, then restart",src, target, target)
			}
			release, _, err := fileutil.Flock(filepath.Join(src, "LOCK"))
			if err != nil {
				return "", nil, err
			}
			log.Info("Migrating instance datadir", "from", src, "to", target)
			if err := os.Rename(src, target); err != nil {
				release.Release()
				return "", nil, err
			}
			return target, release, nil
		}
		if err := os.MkdirAll(target, 0700); err != nil {
			return "", nil, err
		}
	}

	release, _, err = fileutil.Flock(filepath.Join(target, "LOCK"))
	return target, release, err
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
