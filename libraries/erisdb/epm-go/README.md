[![Stories in Ready](https://badge.waffle.io/eris-ltd/deCerver.png?label=ready&title=Ready)](https://waffle.io/eris-ltd/deCerver) || [![GoDoc](https://godoc.org/github.com/epm-go?status.png)](https://godoc.org/github.com/eris-ltd/epm-go) || Master Branch: [![Circle CI](https://circleci.com/gh/eris-ltd/epm-go/tree/master.svg?style=svg)](https://circleci.com/gh/eris-ltd/epm-go/tree/master)

Eris Package Manager: The Smart Contract Package and Blockchain Manager
======

Eris Package Manager, written in `go`. EPM makes it easy to spin up blockchains and to deploy suites of contracts or transactions on them.

At its core is a domain specifc language for specifying contract suite deploy sequences, and a git-like interface for managing multiple blockchains.

For tutorials, see the [website](https://epm.io)

epm-go uses the same spec as the [ruby original](https://github.com/project-douglas/epm), but this is subject to change in an upcoming version. If you have input, please make an issue. We will also soon support yaml and json formatting for deployment suites.

Blockchains
-----------

EPM aims to be chain agnostic, by using `module` wrappers satisfying a [blockchain interface](https://github.com/eris-ltd/epm-go/blob/master/epm/epm.go#L50), built for compatibility with the eris `decerver` ecosystem.
While theoretically any chain can be supported (provided it satisfies the interface), there is currently support for

- `thelonious` (in-process and rpc),
- `ethereum` (in-process),
- `genesisblock` (for deployments of `thelonious` genesis blocks),
- `bitcoin` (through `blockchain.info` api wrapper).

We will continue to add support and functionality as time admits.
If you would like epm to be able to work with your blockchain or software, submit a pull-request to `eris-ltd/modules`
with a wrapper for your chain that satisfies the `Blockchain` interface, as defined in `epm-go/epm/epm.go`.
See the other wrappers in `eris-ltd/modules` for examples and inspiration.

Install
--------

1. [Install go](https://golang.org/doc/install)
2. `go get github.com/eris-ltd/epm-go/cmd/epm`

Formatting
----------
Ethereum input data and storage deals strictly in 32-byte segments or words, most conveniently represented as 64 hex characters.
When representing data, strings are right padded while ints/hex are left padded.

*IMPORTANT*: contracts deployed with `epm` by default use our version of the LLL compiler,
which in addition to adding some opcodes, changes strings to also be left padded.
I repeat, both strings and ints/hex are left padded by default. The reason for this was it simplified the `eris-std-lib`.
Stay tuned for improvement :)

EPM accepts integers, strings, and explicitly hexidecimal strings (ie. "0x45").
If your string is strictly hex characters but missing a "0x", it will still be treated as a normal string, so add the "0x".
Addresses should be prefixed with "0x" whenever possible. Integers in base-10 will be handled, hopefully ok.

Values stored as EPM variables will be immediately converted to the proper hex representation.
That is, if you store "dog", you will find it later as `0x0000000000000000000000000000000000000000000000000000646f67`.

Testing
-------
`go test` can be used to test the parser, or when run in `cmd/tests/` to test the commands.
To test a deployment suite, write a `.pdt` file with the same name as the `.pdx`, where each line consists of query params (address, storage) and the expected result.
A fourth parameter can be included for storing the result as a variable for later queries.
You can test this by running `go run main.go` in `cmd/tests/`.
See [here](`https://github.com/eris-ltd/eris-std-lib/blob/master/DTT/tests/c3d.pdt`) for examples.

Directory
--------
As part of the larger suite of Eris libraries centered on the `deCerver`, epm works out of the core directory in `~/.decerver/blockchains`.
A `HEAD` file tracks the currently active chain and a `refs` directory allows chains to be named.
Otherwise, chains are specified by their `chainId` (signed hash of the genesis block).

Command Line Interface
----------------------
To install the command line tool, cd into `epm-go/cmd/epm/` and hit `go install`.
You can get the dependencies with `go get -d`.
Assuming your `go bin` is on your path, the cli is accessible as `epm`.
EPM provides a git-like interface for managing chains. In general, chains are referred to by their type (short forms allowed) and id (prefix matched), eg. `thel/f852b`, or by a given nickname. We denote the former as `<CHAIN>`, the latter as `<REF>`, and either or as `<ChainRef>`.

Commands:
- `epm init`
    - Initialize the decerver directory tree and default configuration files
- `epm new`
    - Deploy a genesis block from a genesis.json file.
    - The block is saved in a temp folder in ~/.decerver/scratch first, and then copied into the blockchains dir
    - Specify a particular `config.json` and `genesis.json` with the `-config` and `-genesis` flags, respectively.
    - Otherwise, it will use the default `config.json` and `genesis.json`
    - Deploy will automatically open vim for you to edit config files as you deem fit, before saving them to the working directory
- `epm checkout <ChainRef>`
    - Checkout a chain, making it the current working chain.
    - It will be written to the top of the `~/.decerver/blockchains/HEAD` file.
    - Use `epm new -checkout` to checkout a chain immediately after creation.
- `epm fetch <dappname>`
    - Deploy and install a chain from a package.json and genesis.json in a dapp repository
    - Easiest way to sync with a dapp specific chain using just the package.json and genesis.json
- `epm run -chain <ChainRef>`
    - Run a chain. Leaving out the `-chain` flag will default to the HEAD chain.
- `epm deploy <file.pdx>`
    - deploy a `.pdx` file (create contracts, send transactions)
- `epm plop <genesis | config>`
    - Plop the default genesis.json or config.json (respectively) into the current working directory
- `epm [clean | pull | update]`
    - Clean epm related dirs, pull updates to the source code, and re-install the software, respectively.
- `epm refs add <CHAIN> <REF>`
    - Create a new named reference to a chainId
    - Note you can avoid this by using, for example, `epm new -name <name>`, to name the chain during installation
    - Don't forget, `<CHAIN>` means `ChainType/ChainId` (eg. `thel/f852`)
- `epm refs rm <REF>`
    - Remove a reference to a chain
- `epm [refs | head]`
    - Display the available references, or the current head, respectively.
- `epm rm <ChainRef>
    - Remove a chain and all associated data from the decerver tree.
    - Will ask for confirmation
- `epm console`
    - boot epm in interactive mode (like an ipython console) - experimental feature


For more details, see `epm --help` or `epm [command] --help`
