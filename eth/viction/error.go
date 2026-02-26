package viction

import "errors"

var (
	// ErrNoValidator is when the list of validator is empty.
	ErrNoValidator = errors.New("no validator existed")

	// ErrInvalidAttestorList is when the attestors list are not conformed to the consensus rules.
	ErrInvalidAttestorList = errors.New("invalid attestor list")
)
