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

// legacyInstanceDir is the historical instance directory name that may need
// to be migrated to the current instance name (e.g. "viction").
const legacyInstanceDir = "tomo"

// legacyTomoXDir is the TomoX LevelDB directory name.
const legacyTomoXDir = "tomox"

// migrateLegacyInstanceDir resolves the instance directory path and migrates
// the legacy "tomo" directory to targetName when needed
// Returns:
//   - instdir: the resolved instance directory path (always set when no error)
//   - release: non-nil only when migration happened; holds the lock on the
//     moved LOCK file and must be assigned to n.dirLock by the caller
//   - err: non-nil if migration was attempted but failed
func migrateLegacyInstanceDir(dataDir, targetName string) (instdir string, release fileutil.Releaser, err error) {
	instdir = filepath.Join(dataDir, targetName)

	// Skip migration if the target already has node data or if the target
	// name is the legacy name itself.
	if instanceDirHasData(instdir) || targetName == legacyInstanceDir {
		return instdir, nil, nil
	}

	src := filepath.Join(dataDir, legacyInstanceDir)
	if !instanceDirHasData(src) {
		return instdir, nil, nil
	}

	// Guard against renaming over an existing (but empty) target directory.
	if common.FileExist(instdir) {
		return "", nil, fmt.Errorf(
			"[Node] cannot migrate instance datadir from %q to %q: destination already exists; remove or rename %q, then restart",
			src, instdir, instdir,
		)
	}

	release, _, err = fileutil.Flock(filepath.Join(src, "LOCK"))
	if err != nil {
		return "", nil, err
	}

	log.Info("[Node] Migrating instance datadir", "from", src, "to", instdir)
	if err = os.Rename(src, instdir); err != nil {
		return "", nil, err
	}

	if err := migrateLegacyTomoXDir(dataDir, targetName); err != nil {
		return "", nil, err
	}
	return instdir, release, nil
}

// migrateLegacyTomoXDir moves a legacy root-level TomoX database
// ({datadir}/tomox) into the instance directory. It is only called after a
// successful tomo → targetName instance migration.
func migrateLegacyTomoXDir(dataDir, targetName string) error {
	dst := filepath.Join(dataDir, targetName, legacyTomoXDir)
	if tomoxDirHasData(dst) {
		return nil
	}

	src := filepath.Join(dataDir, legacyTomoXDir)
	if !tomoxDirHasData(src) {
		return nil
	}
	if common.FileExist(dst) {
		return fmt.Errorf(
			"[Node] cannot migrate TomoX datadir from %q to %q: destination already exists; remove or rename %q, then restart",
			src, dst, dst,
		)
	}
	log.Info("[Node] Migrating TomoX datadir", "from", src, "to", dst)
	return os.Rename(src, dst)
}

func tomoxDirHasData(dir string) bool {
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
