package viction

import "errors"

var (
	// ErrNoValidator is when the list of validator is empty.
	ErrNoValidator = errors.New("no validator existed")

	// ErrInvalidAttestorList is when the attestors list are not conformed to the consensus rules.
	ErrInvalidAttestorList = errors.New("invalid attestor list")

	// ErrNoContractAddress is when the contract address is not set in the config.
	ErrNoContractAddress = errors.New("contract address is not set")
)
