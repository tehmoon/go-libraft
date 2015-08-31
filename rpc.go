package raft

import (
  "net/http"
  "fmt"
)

// Used to store any field to interact with the RPC
// functions
type RPC struct {
  Server *http.Server
}

// Start the RPC server
func (rpc *RPC) Start(ok chan struct{}) error {
  // TODO: test the server
  go func() {
    ok <- struct{}{}
  }()

  err := rpc.Server.ListenAndServe()

  return err
}

// Create RPC connection
func NewRPC(config *StateMachineConfiguration) (*RPC, error) {
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

  return rpc, nil
}
