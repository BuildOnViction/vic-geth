// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package node

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// legacyInstanceNames lists historical instance directory names under DataDir.
// Order matters: newer names are tried first.
var legacyInstanceNames = []string{"tomo", "geth"}

// migrateLegacyInstanceDir renames a legacy instance directory (tomo, geth) to
// the current instance name (e.g. viction) when the target does not exist yet.
func migrateLegacyInstanceDir(dataDir, targetName string) error {
	if dataDir == "" || targetName == "" {
		return nil
	}
	target := filepath.Join(dataDir, targetName)
	if instanceDirHasData(target) {
		return nil
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
			return fmt.Errorf("legacy datadir %q exists but %q is already present; migrate or remove %q manually", src, target, target)
		}
		log.Info("Migrating instance datadir", "from", src, "to", target)
		return os.Rename(src, target)
	}
	return nil
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
