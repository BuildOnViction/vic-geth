// Copyright 2019 The go-ethereum Authors
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

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"flag"
	"reflect"
	"testing"

	cli "gopkg.in/urfave/cli.v1"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
)

func Test_SplitTagsFlag(t *testing.T) {
	tests := []struct {
		name string
		args string
		want map[string]string
	}{
		{
			"2 tags case",
			"host=localhost,bzzkey=123",
			map[string]string{
				"host":   "localhost",
				"bzzkey": "123",
			},
		},
		{
			"1 tag case",
			"host=localhost123",
			map[string]string{
				"host": "localhost123",
			},
		},
		{
			"empty case",
			"",
			map[string]string{},
		},
		{
			"garbage",
			"smth=smthelse=123",
			map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitTagsFlag(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitTagsFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newEthTestContext(args []string) *cli.Context {
	set := flag.NewFlagSet("setethconfig-test", flag.ContinueOnError)
	for _, f := range []cli.Flag{
		DeveloperFlag,
		LegacyTestnetFlag,
		RopstenFlag,
		RinkebyFlag,
		GoerliFlag,
		YoloV2Flag,
		VictionFlag,
		VictestFlag,
		VicdevFlag,
		NetworkIdFlag,
		SyncModeFlag,
		GCModeFlag,
		TxLookupLimitFlag,
		LegacyLightServFlag,
		LightServeFlag,
		SkipCompatRewindFlag,
	} {
		f.Apply(set)
	}
	if err := set.Parse(args); err != nil {
		panic(err)
	}
	return cli.NewContext(nil, set, nil)
}

func TestSetEthConfigSkipCompatRewind(t *testing.T) {
	// The flag is only set through the CLI; the Viction/Victest datadir default
	// is resolved later in eth.New/les.New by SkipCompatRewindFor.
	tests := []struct {
		name    string
		args    []string
		wantSet bool
		wantVal bool
	}{
		{"unset", nil, false, false},
		{"explicit true", []string{"--skip-compat-rewind"}, true, true},
		{"explicit false", []string{"--skip-compat-rewind=false"}, true, false},
		{"explicit true with network flag", []string{"--viction", "--skip-compat-rewind"}, true, true},
		{"explicit false with network flag", []string{"--victest", "--skip-compat-rewind=false"}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack, err := node.New(&node.Config{DataDir: ""})
			if err != nil {
				t.Fatalf("failed to create test node: %v", err)
			}
			defer stack.Close()

			cfg := eth.DefaultConfig
			SetEthConfig(newEthTestContext(tt.args), stack, &cfg)
			if tt.wantSet {
				if cfg.SkipCompatRewind == nil || *cfg.SkipCompatRewind != tt.wantVal {
					t.Errorf("SkipCompatRewind = %v, want %v", cfg.SkipCompatRewind, tt.wantVal)
				}
			} else if cfg.SkipCompatRewind != nil {
				t.Errorf("SkipCompatRewind = %v, want unset", *cfg.SkipCompatRewind)
			}
		})
	}
}
