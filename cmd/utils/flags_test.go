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
	"time"

	cli "gopkg.in/urfave/cli.v1"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
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

func Test_HTTPTimeoutFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantTimeout time.Duration
	}{
		{"defaultFlag", nil, 120 * time.Second},
		{"explicitValue", []string{"--http.timeout", "300"}, 300 * time.Second},
		{"zeroValueKeepsDefaults", []string{"--http.timeout", "0"}, 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			HTTPTimeoutFlag.Apply(fs)
			if tt.args != nil {
				if err := fs.Parse(tt.args); err != nil {
					t.Fatal(err)
				}
			}
			cfg := node.DefaultConfig

			SetNodeConfig(cli.NewContext(nil, fs, nil), &cfg)

			if cfg.HTTPTimeouts.ReadTimeout != tt.wantTimeout {
				t.Errorf("ReadTimeout = %v, want %v", cfg.HTTPTimeouts.ReadTimeout, tt.wantTimeout)
			}
			if cfg.HTTPTimeouts.WriteTimeout != tt.wantTimeout {
				t.Errorf("WriteTimeout = %v, want %v", cfg.HTTPTimeouts.WriteTimeout, tt.wantTimeout)
			}
			if cfg.HTTPTimeouts.IdleTimeout != rpc.DefaultHTTPTimeouts.IdleTimeout {
				t.Errorf("IdleTimeout = %v, want default %v", cfg.HTTPTimeouts.IdleTimeout, rpc.DefaultHTTPTimeouts.IdleTimeout)
			}
		})
	}
}
