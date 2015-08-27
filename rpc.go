package raft

import (
  "net/http"
  "fmt"
)

type RPC struct {
  Server *http.Server
}

func (rpc *RPC) Start() error {
  err := rpc.Server.ListenAndServe()

  return err
}

func NewRPC() *RPC {
  mux := http.NewServeMux()

  server := &http.Server{
    Addr: ":8081",
    Handler: mux,
  }

  mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello World!")
  })

  rpc := &RPC{
    Server: server,
  }

  return rpc
}
