Eris Consensus Models
=========================
ErisLtd is in the business of maintaining distributed databases upon which business facing applications are run. Some considerations are as follows:

    - Replication: Data should be backed up, in multiple places.
    - Fault-Tolerance: The service must continue to be available in the event of failures
    - Consistency: reads should not yield conflicts. all datastores should remain in sync
    - Non-Repudiation: data and txs should be cryptographically verifiable
    - Latency: users should never have to wait
    

Our Fault-Tolerance Model is :

    - Asynchronous: messages may arrive at any time, be arbitrarily delayed, or dropped
    - Semi-Byzantine: compromise of servers or private keys may create malicious nodes
    - Authenticated: all messages are accompanied by signatures and hashes. Pubkeys are authorized before hand by business specific mechanism.

Many attacks can be mitigated immediately by genesis block permissions. However, compromise of multiple servers can potentially replace the genesis block and chain. 


Stack
-----
    - Client: A javascript browser app
        - users should be able to cache their current account and the app js to their machine (for access during network failure)
    - WebService: eg. enyo.bankofamerica.com
        - serves the client
        - load-balancing, caching, fault-tolerance, standard webapp stuff
        - db backend is a bostejo instance (ie a network of decervers)
    - Bostejo: network of decervers
        - manage the blockchain
        - consensus

WebService
----------
Some Options:

1. The web-service does not run a local decerver:
    - no local copy of the blockchain
    - massively reduced server requirements - offloads everything to the network
    - must have strong caching support
    - staying up-to-date: simplified peer-server connects to many other nodes and keep up with the latest block
        - requires decervers to transmit merkle proofs in query responses and complicates confirmation on the part of the web-service

2. The web-service runs a local decerver:
    - local copy of the blockchain
    - tx processing requirements
    - can do verification locally
    - if this node fails, we are back to 1
    - txs may or may not pass through this node first, but reads come locally
    - participates in consensus
       
3. The web-service has multiple locations
    - the web-service itself may want to be replicated to reduce latencies (mostly read)
    - in this case, it would be expected that each has a local decerver       

It is of course within Eris' scope to deploy and run these web-services, and hence to probably include decervers within them. But a company on a tighter budget can take advantage of bostejo, yet run their own web-server (no local decerver).


Notes on Consensus
------------------
    - we intend to use a model based on blockchains, most particularly for their non-repudiability guarantees
    - this means txs are confirmed strictly in blocks, which introduces confirmation delays
    - in most cases, users can assume their requests are processed immediately:
        - if there is a tx error, the response will be very fast
        - if there is not, the tx may or may not be included in a block
        - if the tx is not quickly included in a block, or another tx is that conflicts with the original one, the user must be notified in a reasonable way. this will be an issue of contention.
            - this may be mitigated easily if there is a local decerver and all txs go through it (unless it fails). Then there's no way things can happen which it does not know about. However, if it fails, we must again rely on the network.
        - most of the time, the request will not fail, but the absolute confirmation still comes later



Note, the eth-go stack gives us a great codebase for managing peer messages, executing smart contracts, and managing the chain, but ethereum has a more severe byzantine context than we do, as they are constantly under attack. We can suppose that the majority of time for eris, an elected leader (or small group of them) *is trustworthy*, especially under randomization. But rather than muck about in paxos-like leader election, we use *stake-weighted proof-of-work* for leader election on a block-by-block basis, where stake can be relegated upfront or dynamically updated by a set of master multi-sig contracts. At no point should the relationship between addresses and ips be made known.

We can go even further with efficiency as follows: 

    - the local web-service decerver is denoted the defacto leader
    - as txs come in, the leader broadcasts them immediately, and commits them in blocks every T seconds
    - blocks are broadcasts to the other decervers, which note the signature and commit the block (they have already seen many of the transactions)
    - decervers are to expect a new block from the leader every T seconds (if no txs come in T seconds, the leader should submit an empty heartbeat block)
    - if T seconds pass before a new block, a decerver will begin PoW mining. This way, we maintain availability, but it slows down somewhat. When the leader is brought back up, it can catch up with the new blocks, and then start broadcasting blocks again.
            - this may be mitigated easily if there is a local decerver and all txs go through it (unless it fails). Then there's no way things can happen which it does not know about. However, if it fails, we must again rely on the network


Notes
-----
    - this model is highly efficient in the absence of failures, and maintains high consistency and availability.
    - security guarantees still depend on genesis permissions
    - writes should regardless be writen/replicated to the local server until there is confirmation that they have been replicated across the network (ie. in the blockchain).
    - if the server fails, commits default to PoW delegated leadership over the network, which slows down writes by perhaps a factor of 10, but keeps us available
    - the server can keep a local master/slave relationship, to give more room before defaulting to the network.
    - Concerns:
        - if the leader is compromised, the system is easily overrun. However, once it is found out, it is easily reverted back to pre-malicious state. Every web-service on the planet suffers from this, but its ok, since they operate mostly in a non-Byzantine context.
        - if the leader fails and commits default to the network, an agent with more than 51% of the decerver hashing power can cause trouble.
            - this is mitigated mostly by signatures, but in the event of a major compromise, is practically indefensible.
                - using very up-to-date asic-hard PoW algorithms is probably our best defence, and we can update them as necessary
                - its also possible to prove a chain is "older" (and therefore more valuable/likely to be correct) by using the bitcoin network:
                    - to timestamp a chain, create a msg for that chain. takes its hash. submit it in a tx to bitcoin
                    - commit the msg, the bitcoin txid, and the bitcoin blockhash for that tx into your chain
                    - you can still pretend your chain is older than it is (by preparing before hand, getting a msg into the bitcoin chain, and starting your chain later), but you can only go as far back as when you thought to begin the attack (which is hopefully much later than the beginning of any eris instance)

        - if the webservice has more than one location, this needs to be heavily adapted
 
