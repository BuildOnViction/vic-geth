// Copyright 2026 The Vic-geth Authors
package core

func (st *StateTransition) applySponsoringTransaction() error {
	st.payer = st.msg.From()
	// TODO: implement sponsoring logic
	return nil
}

func (st *StateTransition) isSponsoringTransaction() bool {
	return st.payer != st.msg.From()
}

func (st *StateTransition) refundGasSponsoringTransaction() {
}
