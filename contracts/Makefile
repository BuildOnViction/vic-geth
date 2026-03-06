# Makefile for generating Go bindings for Viction contracts

ABIGEN = ../build/bin/abigen

.PHONY: all blocksigner validator checkpointoracle zerogas randomize multisigwallet tomox

all: blocksigner validator checkpointoracle zerogas randomize multisigwallet tomox

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

tomox:
	$(ABIGEN) --bin=tomox/contract/build/TRC21.bin --abi=tomox/contract/build/TRC21.abi --pkg=contract --type=TRC21 --out=tomox/contract/TRC21.go
	$(ABIGEN) --bin=tomox/contract/build/TOMOXListing.bin --abi=tomox/contract/build/TOMOXListing.abi --pkg=contract --type=TOMOXListing --out=tomox/contract/TOMOXListing.go
	$(ABIGEN) --bin=tomox/contract/build/RelayerRegistration.bin --abi=tomox/contract/build/RelayerRegistration.abi --pkg=contract --type=RelayerRegistration --out=tomox/contract/RelayerRegistration.go