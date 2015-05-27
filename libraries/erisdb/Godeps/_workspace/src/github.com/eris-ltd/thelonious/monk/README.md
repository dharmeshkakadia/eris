# Monk

--------------------------------------------------
Monk is an interface into thelonious chains that satisfies the `decerver-interfaces` Blockchain interface. The library provides common functions for working with blockchains. It has been designed so that launching a chain from a program takes as little effort as possible:
    
    package main 
    import "github.com/eris-ltd/thelonious/monk"

    func main(){
        m := monk.NewMonk(nil)
        m.Init()
        m.Start()
        m.WaitForShutdown()
    }

All configuration options are run through a single struct, the `ChainConfig`. For example,

        m := monk.NewMonk(nil)
        m.Config.LogLevel = 3
        m.Config.Mining = true
        m.Config.RemoteHost = "192.168.0.10"
        m.Config.ChainName = "my-chain" 
        m.Init()
        m.Start()

will set the log level to 3, start mining, attempt to connect to `192.168.0.10`, and attempt to load the blockchain by name (`my-chain`). See all configuration options and their defaults in `config.go`.
