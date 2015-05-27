# Running 'make' should start a simple go program that runs through the 
# commented imports, turns them on one at a time, maybe has to modify some other things (!), and then compiles that binary. Running epm alone will call os.Exec to call the appropriate binary
#
#
all:
	go install ./cmd/epm-binary-generator
	epm-binary-generator ./cmd/epm ./commands thelonious tendermint ethereum

binary:
	go install ./cmd/epm-binary-generator

tendermint:
	epm-binary-generator ./cmd/epm ./commands tendermint

ethereum:
	epm-binary-generator ./cmd/epm ./commands ethereum

thelonious:
	epm-binary-generator ./cmd/epm ./commands thelonious

