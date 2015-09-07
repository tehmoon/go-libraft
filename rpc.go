package raft

import (
  "net/http"
  "fmt"
  "io/ioutil"
  "strconv"
  "encoding/json"
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
// TODO: create error object for http responses
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
      w.WriteHeader(422)
      return
    }

    // start parsing term
    term, ok := r.Form["term"]
    if !ok || term[0] == "" {
      w.WriteHeader(422)
      return
    }

    termId, err := strconv.ParseUint(term[0], 10, 0)
    if err != nil {
      w.WriteHeader(422)
      return
    }

    if termId < sm.State.CurrentTerm {
      w.WriteHeader(422)
      return
    }
    // end parsing term

    // start parsing prevLogTerm
    prevLogTerm, ok := r.Form["prevLogTerm"]
    if !ok || prevLogTerm[0] == "" {
      w.WriteHeader(422)
      return
    }

    prevTerm, err := strconv.ParseUint(prevLogTerm[0], 10, 0)
    if err != nil {
      w.WriteHeader(422)
      return
    }
    // end parsing prevLogTerm

    // start parsing prevLogIndex
    prevLogIndex, ok := r.Form["prevLogIndex"]
    if !ok || prevLogIndex[0] == "" {
      w.WriteHeader(422)
      return
    }

    prevIndex, err := strconv.ParseUint(prevLogIndex[0], 10, 0)
    if err != nil {
      w.WriteHeader(422)
      return
    }

    // Acquire the lock
    sm.Storage.C <- struct{}{}
    defer func() {
      <- sm.Storage.C
    }()

    // if the previous index of the leader is superior to
    // the number of logs stored on this server and it is not the
    // first log appened then verify if the term entry corresponds
    // to the indexed log, otherwise tell him to retry
    // sending logs but with a lower index
    if prevIndex > sm.Storage.Index {
      w.WriteHeader(416)
      return
    } else if prevIndex != 0 && sm.Storage.Index == 0 {
      w.WriteHeader(416)
      return
    } else if sm.Storage.Index != 0 {
      if sm.Storage.Logs[prevIndex].Term != prevTerm {
        w.WriteHeader(416)
        return
      }
    }
    // end parsing term

    // start parsing body
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      w.WriteHeader(400)
      return
    }

    switch contentType := r.Header.Get("Content-Type"); contentType {
      case "application/json":
        var logs Logs

        err := json.Unmarshal(body, &logs)
        if err != nil {
          w.WriteHeader(400)
          return
        }

        for _, log := range logs {
          sm.Storage.AppendLog(prevIndex, &log)
          prevIndex++
        }

        sm.State.C <- struct{}{}
        sm.State.CurrentTerm = termId
        <- sm.State.C

        fmt.Println(sm.Storage.Logs)
      default:
        w.WriteHeader(415)
        return
    }
    // end parsing body
  })

  rpc := &RPC{
    Server: server,
  }

  return rpc, nil
}
