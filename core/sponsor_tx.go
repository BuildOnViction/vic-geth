// Copyright 2026 The Vic-geth Authors
//
package core

func (st *StateTransition) applySponsoringTransaction() error {
	st.payer = st.msg.From()
	return nil
}
