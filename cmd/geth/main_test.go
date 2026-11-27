// Copyright 2025 The Viction Authors
// (modifications)
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"strings"
	"testing"
)

// Viction networks are logged by name and keep the default cache instead of
// getting the 4 GB bump meant for Ethereum mainnet.
func TestPrepareVictionNetworks(t *testing.T) {
	tests := []struct {
		flag, wantLog string
	}{
		{"--viction", "Starting Geth on Viction mainnet..."},
		{"--victest", "Starting Geth on Viction testnet..."},
	}
	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			geth := runGeth(t, tt.flag, "--port", "0", "--nat", "none", "--nodiscover", "--maxpeers", "0", "--ipcdisable",
				"console", "--exec", "1")
			geth.WaitExit()
			logs := geth.StderrText()
			if !strings.Contains(logs, tt.wantLog) {
				t.Errorf("log %q not found", tt.wantLog)
			}
			for _, bad := range []string{"Starting Geth on Ethereum mainnet", "Bumping default cache on mainnet"} {
				if strings.Contains(logs, bad) {
					t.Errorf("unexpected log %q", bad)
				}
			}
		})
	}
}
