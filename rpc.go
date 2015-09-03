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
func (rpc *RPC) Start() error {
  // TODO: test the server
  go func() {
  }()

  err := rpc.Server.ListenAndServe()

  return err
}

// Create RPC connection
func NewRPC(sm *StateMachine) (*RPC, error) {
  mux := http.NewServeMux()

  addr := fmt.Sprintf("%s:%d", sm.Configuration.RPCHost, sm.Configuration.RPCPort)

  server := &http.Server{
    Addr: addr,
    Handler: mux,
  }

  mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello World!")
  })
  mux.HandleFunc("/appendEntry", func(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
      w.WriteHeader(406)
      return
    }

    // TODO: Parse payload
  })

  rpc := &RPC{
    Server: server,
  }

  return rpc, nil
}
