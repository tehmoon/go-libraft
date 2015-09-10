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
  StateMachine *StateMachine
}

// Start the RPC server
func (rpc *RPC) Start() error {
  // TODO: test the server
  go func() {
  }()

  err := rpc.Server.ListenAndServe()

  return err
}

func (rpc *RPC) StartElection() {
  sm := rpc.StateMachine

  if sm.State.Is() != CANDIDATE {
    return
  }

  sm.State.SyncTerm <- struct{}{}
  defer func() {
    <- sm.State.SyncTerm
  }()

  oldTerm := sm.State.CurrentTerm
  newTerm := sm.State.CurrentTerm + 1

  sm.State.CurrentTerm = newTerm
  sm.Exec("term::changed", oldTerm, newTerm)

  if ok := sm.Timer.Stop(); ok {
    for ok := false; !ok; {
      ok = sm.Timer.Start()
    }
  }

  sm.State.Switch(LEADER)
  sm.Timer.Stop()
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
    // Declare a variable that will be used to be the statusCode of the response
    // because WriteHeader actually writes header, it is not possible to play
    // with the header in a defer function's call
    statusCode := 200

    // Acquire the lock since we'll write and we don't want dirty reads
    sm.State.SyncTerm <- struct{}{}
    defer func() {
      // Send back the Current-Term in the response
      w.Header().Add("X-Current-Term", strconv.FormatUint(sm.State.CurrentTerm, 10))
      w.WriteHeader(statusCode)

      // Unlock
      <- sm.State.SyncTerm
    }()

    oldTerm := sm.State.CurrentTerm

    err := r.ParseForm()
    if err != nil {
      statusCode = 422
      return
    }

    // start parsing term
    term, ok := r.Form["term"]
    if !ok || term[0] == "" {
      statusCode = 422
      return
    }

    newTerm, err := strconv.ParseUint(term[0], 10, 0)
    if err != nil {
      statusCode = 422
      return
    }

    if newTerm < sm.State.CurrentTerm {
      statusCode = 422
      return
    }
    // end parsing term

    //// start parsing leaderCommit
    //leaderCommitValue, ok := r.Form["leaderCommit"]
    //if !ok || leaderCommitValue[0] == "" {
      //statusCode = 422
      //return
    //}

    //leaderCommit, err := strconv.ParseInt(leaderCommitValue[0], 10, 0)
    //if err != nil {
      //statusCode = 422
      //return
    //}
    //// end parsing leaderCommit

    //// start parsing prevLogTerm
    //prevLogTerm, ok := r.Form["prevLogTerm"]
    //if !ok || prevLogTerm[0] == "" {
      //statusCode = 422
      //return
    //}

    //prevTerm, err := strconv.ParseUint(prevLogTerm[0], 10, 0)
    //if err != nil {
      //statusCode = 422
      //return
    //}
    //// end parsing prevLogTerm

    //// start parsing prevLogIndex
    //prevLogIndex, ok := r.Form["prevLogIndex"]
    //if !ok || prevLogIndex[0] == "" {
      //statusCode = 422
      //return
    //}

    //prevIndex, err := strconv.ParseInt(prevLogIndex[0], 10, 0)
    //if err != nil {
      //statusCode = 422
      //return
    //}

    //// Acquire the lock
    //sm.Storage.C <- struct{}{}
    //defer func() {
      //<- sm.Storage.C
    //}()

    //// end parsing term

    // start parsing body
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      statusCode = 400
      return
    }

    switch contentType := r.Header.Get("Content-Type"); contentType {
      case "application/json":
        var logs Logs

        err := json.Unmarshal(body, &logs)
        if err != nil {
          statusCode = 400
          return
        }

        if sm.State.Is() != FOLLOWER {
          sm.State.Switch(FOLLOWER)
        }

        if len(logs) == 0 && newTerm != sm.State.CurrentTerm {
          sm.State.CurrentTerm = newTerm
          sm.Exec("term::changed", oldTerm, newTerm)
        }

        // TODO: stop the timer after added all logs
        sm.Timer.Stop();
        for ok := false; !ok; {
          ok = sm.Timer.Start()
        }
      default:
        statusCode = 415
        return
    }
    // end parsing body

  })

  rpc := &RPC{
    Server: server,
    StateMachine: sm,
  }

  return rpc, nil
}
