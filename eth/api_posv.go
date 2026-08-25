// Copyright 2015 The go-ethereum Authors
// (original work)
// Copyright 2025 The Viction Authors
// (modifications)
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth

type PublicPosvAPI struct {
	e *Ethereum
}

// NewPublicPosvAPI creates a new PoSV consensus API for full nodes.
func NewPublicPosvAPI(e *Ethereum) *PublicPosvAPI {
	return &PublicPosvAPI{e}
}

type PublicPosvDebugAPI struct {
	e *Ethereum
}

// NewPublicPosvDebugAPI creates a new PoSV consensus debug API for archive nodes.
func NewPublicPosvDebugAPI(e *Ethereum) *PublicPosvDebugAPI {
	return &PublicPosvDebugAPI{e}
}
