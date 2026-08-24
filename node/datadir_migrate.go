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

const legacyInstanceDir = "tomo"
const legacyTradingDir = "tomox"

func migrateLegacyInstanceDir(dataDir, targetName string) (instdir string, release fileutil.Releaser, err error) {
	instdir = filepath.Join(dataDir, targetName)

	if instanceDirHasData(instdir) || targetName == legacyInstanceDir {
		return instdir, nil, nil
	}

	src := filepath.Join(dataDir, legacyInstanceDir)
	if !instanceDirHasData(src) {
		return instdir, nil, nil
	}

	if common.FileExist(instdir) {
		return "", nil, fmt.Errorf(
			"cannot migrate instance datadir from %q to %q: destination already exists; remove or rename %q, then restart",
			src, instdir, instdir,
		)
	}

	release, _, err = fileutil.Flock(filepath.Join(src, "LOCK"))
	if err != nil {
		return "", nil, err
	}

	log.Info("Migrating instance datadir", "from", src, "to", instdir)
	if err = os.Rename(src, instdir); err != nil {
		return "", nil, err
	}

	if err := migrateLegacyTradingDir(dataDir, targetName); err != nil {
		return "", nil, err
	}
	return instdir, release, nil
}

func migrateLegacyTradingDir(dataDir, targetName string) error {
	dst := filepath.Join(dataDir, targetName, legacyTradingDir)
	if tradingDirHasData(dst) {
		return nil
	}

	src := filepath.Join(dataDir, legacyTradingDir)
	if !tradingDirHasData(src) {
		return nil
	}
	if common.FileExist(dst) {
		return fmt.Errorf(
			"cannot migrate Trading datadir from %q to %q: destination already exists; remove or rename %q, then restart",
			src, dst, dst,
		)
	}
	log.Info("Migrating Trading datadir", "from", src, "to", dst)
	return os.Rename(src, dst)
}

func tradingDirHasData(dir string) bool {
	for _, name := range []string{"CURRENT", "LOCK"} {
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
