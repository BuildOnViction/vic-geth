# Makefile for generating Go bindings for Viction contracts

ABIGEN = ../build/bin/abigen

.PHONY: all blocksigner validator checkpointoracle zerogas randomize multisigwallet

all: blocksigner validator checkpointoracle zerogas randomize multisigwallet

blocksigner:
	$(ABIGEN) --bin=blocksigner/contract/build/BlockSigner.bin --abi=blocksigner/contract/build/BlockSigner.abi --pkg=contract --type=BlockSigner --out=blocksigner/contract/viction_block_signer.go

validator:
	$(ABIGEN) --bin=validator/contract/build/Validator.bin --abi=validator/contract/build/Validator.abi --pkg=contract --type=Validator --out=validator/contract/viction_validator.go

zerogas:
	$(ABIGEN) --bin=zerogas/contract/build/ZeroGas.bin --abi=zerogas/contract/build/ZeroGas.abi --pkg=contract --type=ZeroGas --out=zerogas/contract/viction_zero_gas.go

randomize:
	$(ABIGEN) --bin=randomize/contract/build/Randomize.bin --abi=randomize/contract/build/Randomize.abi --pkg=contract --type=Randomize --out=randomize/contract/viction_randomize.go

multisigwallet:
	$(ABIGEN) --bin=multisigwallet/contract/build/MultiSigWallet.bin --abi=multisigwallet/contract/build/MultiSigWallet.abi --pkg=contract --type=MultiSigWallet --out=multisigwallet/contract/viction_multisig_wallet.go