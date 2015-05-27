[![GoDoc](https://godoc.org/github.com/eris-ltd/lllc-server?status.png)](https://godoc.org/github.com/eris-ltd/lllc-server)

[![Circle CI](https://circleci.com/gh/eris-ltd/lllc-server.svg?style=svg)](https://circleci.com/gh/eris-ltd/lllc-server)

lllc-server
===========

The Lovely Little Language Compiler: A web server and client for compiling ethereum languages.

# Features

- language agnostic (currently supports lll, serpent2.0, solidity)
- returns ethereum abis (for serpent and solidity)
- handles included files recursively with regex matching
- client side and server side caching
- configuration file with per-language options
- local proxy server for compiling from languages other than go
- easily extensible to new languages

Eris Industries' own public facing LLLC-server (at http://lllc.erisindustries.com) is hardcoded into the source,
so you can start compiling ethereum language right out of the box with no extra tools required.

If you want to use your own server, or default to compiling locally, or otherwise adjust configuration settings,
see the config file at `~/.eris/languages/config.json`.

# How to play

## Using the Golang API

```
bytecode, err := lllcserver.Compile("mycontract.lll")
```

The language is determined automatically from extension. If you want to compile literal expressions,
you must specify the language explicitly, ie.

```
bytecode, err := lllcserver.CompileLiteral("[0x5](+ 4 @0x3)", "lll")
```

## Using the CLI

#### Compile Remotely

```
lllc-server compile --host http://lllc.erisindustries.com:8090 test.lll
```

Leave out the `--host` flag to default to the url in the config.

#### Compile Locally
Make sure you have the appropriate compiler installed and configured (you may need to adjust the `cmd` field in the config file)

```
lllc-server compile --local test.lll
```

#### Run a server yourself

```
lllc-server --port 9000
```

## Using the json-rpc proxy server

If you are coding in another language and would like to use the lllc-server client without wrapping the command line, run a proxy server and send it a simple http-json request.

To run the proxy:

```
lllc-server proxy --port 9099
```

And the JSON request:

```
{
 source:"myfile.se"
}
```

Or, to compile literals:

```
{
 source: "x=5",
 literal: true,
 language: "se"
}
```

The response JSON looks like:

```
{
 bytecode:"600580600b60003960105660056020525b6000f3",
 error:""
}
```

To test, stick one of the above JSON requests into `file.json` and run

```
curl -X POST -d @file.json http://localhost:9099 --header "Content-Type:application/json"
```

# Install

The lllc-server itself can be installed with

```
go get github.com/eris-ltd/lllc-server/cmd/lllc-server
```

Installing the actual compilers is a bit more involved. You need a bunch of cpp dependencies :(

See [ethereum wiki](https://github.com/ethereum/cpp-ethereum/wiki/Building-on-Ubuntu) for dependencies (no need for qt)

Note, thelonious and its Genesis Doug were build on a previous version of the languages (before the ABI spec) and so currently only support PoC6 LLL and Serpent 1.0.
But epm works fine using Solidity and Serpent on standard ethereum chains.

To install `LLL` (eris LLL, which includes a few extra opcodes)

```
git clone git@github.com:eris-ltd/eris-cpp
cd eris-cpp/build
bash instructions
```

Install `Serpent`:
```
git clone git@github.com:ethereum/serpent
cd serpent
make
sudo make install
```

Install `Solidity`:

```
git clone git@github.com:ethereum/cpp-ethereum
cd cpp-ethereum
mkdir build
cd build
cmake .. -
make -j2
```

Now the final thing is make sure the configuration paths are properly set.
Running `epm init` (assuming epm is installed) should create the config file at `~/.eris/languages/config.json`.
Edit the `cmd` field for each language to have the correct path.

# Support

Run `lllc-server --help` or `lllc-server compile --help` for more info, or come talk to us on irc at #erisindustries and #erisindustries-dev.

If you are working on a language, and would like to have it supported, please create an issue! Note it is possible to add new languages simply by editing the config file, without having to recompile the lllc-server source code.

