# Makefile for generating Go bindings for Viction contracts

ABIGEN = ../build/bin/abigen

.PHONY: all blocksigner checkpointoracle multisig randomize token trading validator zerogas

all: blocksigner checkpointoracle multisig randomize token trading validator zerogas

blocksigner:
	$(ABIGEN) --bin=blocksigner/contract/build/BlockSigner.bin --abi=blocksigner/contract/build/BlockSigner.abi --pkg=contract --type=BlockSigner --out=blocksigner/contract/viction_block_signer.go

multisig:
	$(ABIGEN) --bin=multisig/contract/build/MultisigWallet.bin --abi=multisig/contract/build/MultisigWallet.abi --pkg=contract --type=Multisig --out=multisig/contract/viction_multisig_wallet.go

randomize:
	$(ABIGEN) --bin=randomize/contract/build/Randomize.bin --abi=randomize/contract/build/Randomize.abi --pkg=contract --type=Randomize --out=randomize/contract/viction_randomize.go

token:
	$(ABIGEN) --bin=token/contract/build/FungibleToken.bin --abi=token/contract/build/FungibleToken.abi --pkg=contract --type=FungibleToken --out=token/contract/fungible_token.go

trading:
	$(ABIGEN) --bin=trading/contract/build/TradingListing.bin --abi=trading/contract/build/TradingListing.abi --pkg=contract --type=TradingListing --out=trading/contract/viction_trading_listing.go
	$(ABIGEN) --bin=trading/contract/build/RelayerRegistration.bin --abi=trading/contract/build/RelayerRegistration.abi --pkg=contract --type=RelayerRegistration --out=trading/contract/viction_relayer_registration.go

validator:
	$(ABIGEN) --bin=validator/contract/build/Validator.bin --abi=validator/contract/build/Validator.abi --pkg=contract --type=Validator --out=validator/contract/viction_validator.go

zerogas:
	$(ABIGEN) --bin=zerogas/contract/build/ZeroGas.bin --abi=zerogas/contract/build/ZeroGas.abi --pkg=contract --type=ZeroGas --out=zerogas/contract/viction_zero_gas.go
