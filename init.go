package raft

import (
  "runtime"
  "log"
)

func init() {
  runtime.GOMAXPROCS(runtime.NumCPU())
  log.SetFlags(log.Lmicroseconds)
}
