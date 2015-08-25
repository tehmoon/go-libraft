package raft

import (
  "net/http"
)

type RPC struct {
  Server *http.Server
}

func (rpc *RPC) Start() {
  rpc.Server.ListenAndServe()
}

func NewRPC() *RPC {
  server := &http.Server{
    Addr: ":8080",
  }

  rpc := &RPC{
    Server: server,
  }

  return rpc
}
