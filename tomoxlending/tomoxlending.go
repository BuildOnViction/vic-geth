package tomoxlending

import (
	"github.com/ethereum/go-ethereum/tomoxlending/lendingstate"
)

type Lending struct{}

func (l *Lending) GetStateCache() lendingstate.Database {
	return nil
}
