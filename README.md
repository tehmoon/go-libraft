# go-libraft
Implementation of the Raft consensus algorithm in GO.

This is only a lib, it will try to respect as much as possible the Raft understanding and safety policy so people can use it to build anything on top of it.

ie:
 - Raft as a service (zookeeper)
 - A new database
 - Some software with Raft embedded
 - An API
 - A game
 - ...

## Features:

- [ ] Leader Election
- [ ] Randomized Election Timeouts
- [ ] Leader Heartbeat Broadcasting
- [ ] Cluster Membership Change
- [ ] Safe Disk Commits For Logs
- [ ] Index Based Log Replication
- [ ] Snapshot Based Log Replication
- [ ] Log Snapshoting
- [ ] Log Compactation
- [ ] json http RPCs

## Possible Enhancements

- [ ] Safe Channel between RPCs / TLS or PGP ?
- [ ] HTTP2 / HTTP1.1 bi-directionnal com WebSocket ?
- [ ] Fast RPCs / protobuf ?
- [ ] Embedded Database / RocksDB ?

## Disclamer

This project is under heavy developpement, it is part of my way to understand consensus algorithms AND not suitable for any production use.

Issues are welcomed but I'll (at the moment) not accept any PR. Once all the basic features will work I'll be happy to welcome everyone to participate.
