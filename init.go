package raft

import "runtime"

func init() {
  runtime.GOMAXPROCS(runtime.NumCPU())
}
