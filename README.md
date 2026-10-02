# Vic Geth

Golang implementation of Viction Blockchain.

## About Viction

Viction Blockchain (or Viction for short) is an innovative solution to the scalability problem with the Ethereum blockchain. Our mission is to be a leading force in building the Internet of Value, and its infrastructure. We are working to create an alternative, scalable financial system which is more secure, transparent, efficient, inclusive, and equitable for everyone.

Viction relies on a system of 150 Masternodes with a Proof of Stake Voting consensus that can support near-zero fee, and 2-second transaction confirmation times. Security, stability, and chain finality are guaranteed via novel techniques such as double validation, staking via smart-contracts, and "true" randomization processes.

Viction supports all EVM-compatible smart-contracts, protocols, and atomic cross-chain token transfers. New scaling techniques such as sharding, private-chain generation, and hardware integration will be continuously researched and incorporated into Viction Blockchain's masternode architecture. This architecture will be an ideal scalable smart-contract public blockchain for decentralized apps, token issuances, and token integrations for small and big businesses.

More details can be found at our [white papers](https://docs.viction.xyz/whitepaper-and-research).

## Getting started

### For node operators

- Binaries releases can be found in the [releases](releases) page.

- Docker image can be found here: [buildonviction/vic-geth](https://hub.docker.com/r/buildonviction/vic-geth).

### For developers

- Checkout this repository.

- Build the project using Go `v1.18` up to `v1.22`. The compiled binaries can be found in `build/bin` folder.

```
make all
```

- Other helpful commands:

```
make lint   // Enforce coding convention/styling
make test   // Run all tests
make clean  // Clear go cache
```

## Documentation & Resources

- Documentation: [https://docs.viction.xyz/](https://docs.viction.xyz/)
- Homepage: [https://viction.xyz/](https://viction.xyz/)
- Block Explorer: [https://www.vicscan.xyz/](https://www.vicscan.xyz/)
- Validators Governance: [https://www.vicmaster.xyz/](https://www.vicmaster.xyz/)
- Public RPC Service: [https://rpc.viction.xyz/](https://rpc.viction.xyz/)

## Contribution

Thank you for considering to help out with the source code! We welcome contributions
from anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to go-ethereum, please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base. If you wish to submit
more complex changes though, please check up with the core devs first on [our gitter channel](https://gitter.im/ethereum/go-ethereum)
to ensure those changes are in line with the general philosophy of the project and/or get
some early feedback which can make both your efforts much lighter as well as our review
and merge procedures quick and simple.

Please make sure your contributions adhere to our coding guidelines:

 * Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting)
   guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
 * Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary)
   guidelines.
 * Pull requests need to be based on and opened against the `master` branch.
 * Commit messages should be prefixed with the package(s) they modify.
   * E.g. "eth, rpc: make trace configs optional"

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.
