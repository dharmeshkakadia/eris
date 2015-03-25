---

layout: default
title: "Eris Package Manager: Tutorials"

---

# Intro

EPM is a library for deploying suites of smart contracts, and a command line tool for managing multiple blockchains.

EPM aims to be chain agnostic, and to provide either inprocess or rpc communication with a blockchain client through modules
that satisfy the [blockchain interface](https://github.com/eris-ltd/epm-go/blob/cli/epm/epm.go). This tutorial will guide you through the following:

- [Installation](#install)
- [Initialization](#init)
- [QuickStart](#quickstart)
- [Config](#config)
- [Managing chains: new, refs, checkout, head, rm](#chains)
- [Run a thelonious chain](#run-a-thelonious-chain)
- [Transact on a chain](#transact-on-a-chain)
- [Smart contract packages](#smart-contract-packages)
- [Rpc-based interaction with a chain (local or remote)](#rpc-with-thelonious)
- [Using other chains (eg. ethereum, bitcoin)](#using-other-chains)

# Install
Make sure [go is installed](https://golang.org/doc/install). Now

```
go get github.com/eris-ltd/epm-go/cmd/epm
```

If `$GOPATH/bin` is on your path, the cli is available as `epm`. See `epm --help` for details.

To update in the future:

```
cd $GOPATH/src/github.com/eris-ltd/epm-go/cmd/epm
go get -d
go install
```

# Init
Note epm has just received a major overhaul and is somewhat backwards incompatible with the first released version. 
If you have already played with any eris tools, you should delete `~/.decerver`, or at least `~/.decerver/blockchains`
before continuing.

The first thing to do is set up the global decerver directory tree at `~/.decerver`. Do this by running 

```
epm init
```

If ever you feel things have gone wrong, try removing `~/.decerver` and rerunning `epm init`. If you ever remove something within `~/.decerver`, `epm init` ought to restore it for you. 

WARNING! Data is not backed up (yet). Do not remove directories if you are working with private keys protecting real wealth, unless you back them up yourself.

Managing Chains
---------------

# QuickStart

EPM uses chainIds to manage chains, where an id is typically a hash of a signed genesis hash. 
Chains can be given names for easy reference, and a `HEAD` chain can be set, such that EPM defaults to using that chain.

To deploy a new chain, name it, and check it out as the head, run 

```
epm new -name chain1 -checkout
```

Note two vim windows will pop up - for now just close them both with `:wq` twice.

The final line of log output should contain the chainId. 

Now that you have the chain installed, you can run a full node on it with

```
epm --log 3 run
```

Congratulations. You just deployed and ran a thelonious chain.

Exit with `CTRL-C`.

Prefer ethereum?

```
epm new -name ethchain -checkout -type eth
epm --log 3 run
```

Nice, ye?

Most of the rest of the tutorial is in reference to the thelonious chain, but it should all work for ethereum too.
To get back on the thelonious chain, check it out:

```
epm checkout chain1
```

# Config

You can see the contents of a chain's directory at `~/.decerver/blockchains/thelonious/<chainId>`. 
In particular note the `config.json` (this was the second vim 
window that popped up). These values can be modified with the `epm config` command:

```
epm config local_port:30305
epm config mining:true
epm config client:epm-tutorial
```

Or, all at once:

```
epm config local_port:30305 mining:true client:epm-tutorial
```

You can boot it into vim with

```
epm config -vi
```

The chain's directory also contains a `genesis.json` (the first vim window that popped up on deploy),
but it is rather of sentimental or referential value, and should not be changed (nor should changing it affect
anything). All the information from genesis.json is written into the blockchain database in the form of the genesis
block during deployment.


# Chains

You can view the current HEAD (current working chain) with

```
epm head
```

It should give you the chainId you just deployed. Commands like `epm run`, `epm config`, and `epm cmd` default
to using the HEAD, but can be told to use a different chain using the `-chain` flag. The `-chain` flag can either 
specify a named reference or a `<chainType>/<chainId>`. Note the chain id must be preceded by the chain type!

The list of names (references) can be seen with 

```
epm refs
```

The current head should be emphasized. A new ref can be added anytime with

```
epm refs add <chainType>/<chainId> <new name>
```

where `<chainType>` is the type of chain (`thel`, `eth`, etc.), `<chainId>` is a prefix of some known chainId, and `<new name>` is the new reference.

Similarly, remove a ref with 

```
epm refs rm <ref>
```

A given chain can be checked out as HEAD with 

```
epm checkout <chain>
```

where `<chain>` may be a `<chainType>/<chainId>` or named reference.

# New

Note the original deploy sequence can be broken down:

```
epm new
```

The two vim windows that pop up are for the `config.json` and the `genesis.json`, respectively.
You can find more details about the `genesis.json` in the [thelonious tutorial](https://thelonious.io/tutorials/).
For now, I will simply have you note that if you want to deploy multiple thelonious chains with the same settings you will 
need to change the `unique` field to true in the `genesis.json` during the deploy (the second vim window that 
pops up).

A genesis block is deployed from the `genesis.json` into a temporary folder in `~/.decerver/scratch`.
Both the `config.json` and `genesis.json` are copied in as well.
The chainId is then set, and the database and json files are moved from the temporary directory into the 
appropriate location in `~/.decerver/blockchains`.
You can then check out or add a reference for the chain later:

You can of course add the reference right away but checkout later with

```
epm new -name newref
epm checkout newref
```

or do it all at once with

```
epm new -name newref -checkout
```

# Refs

Chains are referred to either by a reference name or by their chainType/chainId, or else not at all, in which case epm defaults to the HEAD chain (`epm head`).
To summarize:

```
<chain> = <chainType/chainId> or <chainRef>
```

Note `<chainId>` always accepts prefixes, and `<chainType>` allows short forms (eg. `eth` or `ethereum`, `thel` or `thelonious` or `monk`)

So for example you can checkout the ethereum chain we made with

```
epm checkout eth/def
```

or with 

```
epm checkout ethchain
```

For commands which don't take the chain as an argument, specify with the `-chain` flag, or else it will default to the current HEAD.

You can delete a chain anytime with

```
epm rm <chain>
```

eg. `epm rm chain1`. Note this removes the entire directory and the references, not just the references. 
If you only want to delete a reference, use

```
epm refs rm <ref>
```

Run `--help` on any of the commands to see more information.

# Run a thelonious chain

Once you have a chain deployed, installed, and checked out (eg. `epm new -checkout`),
simply run 

```
epm --log 3 run
```

to run a full node for that chain. If you don't have a chain checked out or you want to run a different chain, use

```
epm --log 3 run -chain <chain>
```

Note the `log level` defaults to `2`, which shows you the bare minimum useful information when you execute commands. 
For the `run` command, you won't see anything unless you use a higher log level.

# Run a (thelonious) dapp

Running a dapp amounts to getting some parameters (a `package.json` and `genesis.json` file) and installing a chain from them.
By dapp, we really mean a suite of smart contracts. Currently, we follow a one-dapp-one-chain paradigm, but we will become more flexible in the future.

We are still working on streamlining this process. In the meantime, see our [dapp tutorial](https://decerver.io/tutorials/).

# Transact on a chain

The [EPM spec](https://epm.io) defines the commands available to EPM for interacting with a chain.
For example, let's create a contract, deploy it, and then send it a message.

First, create a new working directory and cd into it:

```
mkdir epmtut
cd epmtut
```

Now, create a file `contract.lll` with contents:

```
{
    (return 0 (lll {
        [[(CALLDATALOAD 0x0)]] (CALLDATALOAD 0x20)
    } 0))
}
```

This is a dead simple name registry, but it's perfect for trying out epm commands.

Now run

```
epm cmd deploy contract.lll {{c}}
epm cmd transact {{c}} "0x5 0xf"
epm cmd query {{c}} 0x5 result
```

The `{{}}` denote persistent variables, so here we deploy contract.lll and save
the contract address as {{c}}. Then we can interact with {{c}} through a 
transaction, in this case with data `0x5` and `0xf`. The contract should store
`0xf` at `0x5`. We can check with a query. It should return `0xf`.

Cool, right?!

# Smart Contract Packages

To deploy suites of smart contracts and/or send many transactions, use a `.pdx` file (package definition executuable - but it's not really executable). 
Pdx files use the same commands as `epm cmd`. You can find more details at the [epm docs](https://epm.io).

For example, let's take the previous commands and write them to `tutorial.pdx`:

```
deploy:
    contract.lll => {{c}}
transact:
    {{c}} => 0x5 0xf
```

Note the extra `=>` notation - a vestige of epm's ruby heritage. Also we have removed the quotes around the transact arguments.

Further, we leave out the `query` command as it has no effect until transactions are committed in a block. 
Commits happen after each command from the cli, but in a `.pdx` they only happen at the end. 
Fortunately, you don't really need queries, because you can use tests instead.

To test the deployment, include a `.pdt` file in the same directory as the `.pdx`. 
Each line of a `.pdt` file specifies a test and should have the form

```
contract_addr; storage_addr; expected_result; {{var}}
```

If the value stored at `storage_addr` in the contract at `contract_addr` does not equal
`expected_result`, the test fails. The result of the test can be (optionally) 
stored via `{{var}}` for use in later tests. Note you can grab values but not test them
by setting `expected_result` to `_`.
See an example [here](https://github.com/eris-ltd/eris-std-lib/blob/master/DTT/tests/double.pdt).

In our case, we could write `tutorial.pdt`:

```
{{c}}; 0x5; 0xf
```

And run the deployment again:

```
epm deploy tutorial.pdx
```

Your test should pass with flying colors.

By default, epm will look for contracts in the current directory, 
but use the `-c` flag to set the contract root to another directory.

WARNING: to preserve import paths, the entire contents of the contract directory 
is copied into a cache, so the contract folder ought not contain more than the contracts themselves.
This is why we created the folder `epmtut` before. If you instead kept the contracts
in your home directory, say, epm would attempt to copy your entire home directory. 
That would obviously be bad - it's a subtle point, but if you ever find epm hanging, this might be why!

# Rpc with thelonious

It's often the case that you have a blockchain client already running, and would like to now interact with it using its rpc server.

First, you'll want to make sure the rpc server is on. We can edit a chain's config with `epm config`:

```
epm config serve_rpc:true
epm config rpc_port:30304
epm config rpc_host:"localhost"
epm config mining:true
```

Of course, you can do it all at once: `epm config serve_rpc:true rpc_port:30304 rpc_host:"localhost" mining:true`

Run the chain with 

```
epm run
```

Now, in a new terminal, `cd` to the folder in which you saved `contract.lll`.
To deploy the contract over rpc:

```
epm --rpc --port 30304 cmd deploy contract.lll {{c}}

```

You should see the result in the other terminal window.

You can also deploy `.pdx` files over rpc:

```
epm --rpc --port 30304 deploy tutorial.pdx
```

Using rpc means using the `monkrpc` module wrapper and attempting to connect via HTTP 
on the given host port.

You should see the results of rpc based calls in the other terminal window (assuming the log level is high enough)

Note you can also deploy and install an RPC chain. The purpose of this is simply so you can save a config file 
and not add the host/port or other flags every time. To do so:

```
epm --rpc new
```

And then edit the config as you would previously. Rpc chains are not standalone in that you can not check them out or give them 
a chain ID per se, rather, they piggy back on some other installed chain and require it to be running. 
So if you have a chain already checked out (and have turned on the rpc server in its config and have it running) 
then simply add the `--rpc` flag to any commands and they should read from the 
installed rpc config (assuming you did `epm --rpc new`) and pipe commands over HTTP.

Note, finally, that once you use `--rpc --port 30304` once, you can drop the `--port` flag on subsequent calls, since it will 
automatically save a config file for you in `~/.decerver/blockchains/<chainType>/<chainId>/rpc`. That way you can just
avoid `epm --rpc new` altogether.

# Using Other Chains

EPM was built to be chain agnostic. You want an ethereum chain? Easy:

```
epm new -type eth -checkout
epm head
```

Everything else is the same. I have given the ethereum chain the default chainID of `default` but will update it over time.
You can check it out (`epm checkout eth/default`, if you don't use the `-checkout` flag on `new`) 
and work on it as you would a thelonious chain, deploying contracts and sending transactions.

Note that unlike with a thelonious chain, you cannot transact right away, since you will not have a key in the genesis block.
So you must mine first to get yourself some ether:

```
epm config mining:true
epm --log 3 run
```

Then once you've mined some blocks, kill the process and

```
epm deploy tutorial.pdx
```

The ethereum rpc module hasn't been implemented just yet, but soon. In principle bitcoin is available through the blockchain.info module
wrapper (`epm new -type btc`), but is currently disabled as we have more testing to do.


# Multiple versions of a chain

I have found it useful to be able to work with multiple copies of the same chain (ie. same ChainId), usually for testing purposes.

This can be achieved using the `epm cp` command to duplicate a chain and the `-multi` flag to refer to duplicates. 


For example, to make a copy of the `thel/f8` chain called "copy":

```
epm cp thel/f8 copy
```

We can then run it with 

```
epm run --chain thel/f8 --multi copy
```

Or, if `thel/f8` is already checked out,

```
epm run --multi copy
```

Note that duplicates *must* be named (but feel free to use numbers), and cannot be given refs or checked out directly.
Chains can be found under `~/.decerver/blockchains/<chainType>/<chainId>/<multi name>`. 
A The default chain for a given ChainId has `<multi name> = 0`


# Conclusion

And that's that! Merry blockchaining :)

# Support

Issues and Pull Requests encouraged at https://github.com/eris-ltd/epm-go

Find us on irc at #erisindustries

