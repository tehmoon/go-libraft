package raft

import (
  "net/http"
  "fmt"
  "time"
  "io/ioutil"
  "strconv"
  "encoding/json"
  "net/url"
  "bytes"
  "log"
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

  sm.Timer.Stop();

  oldTerm := sm.State.CurrentTerm
  newTerm := sm.State.CurrentTerm + 1

  sm.State.CurrentTerm = newTerm
  sm.Exec("term::changed", oldTerm, newTerm)

  sm.State.VotedFor = sm.State.MyId

  timer := time.NewTimer(300 * time.Millisecond)

  addOkNode := make(chan struct{}, len(sm.Cluster.Nodes))
  // we start at one because we implicitly count this statemachine
  okNode := 1
  done := make(chan struct{}, 0)

  // counter of node who positivly responded to request-vote
  go func() {
    for ; okNode != len(sm.Cluster.Nodes); okNode++ {
      <- addOkNode
    }

    done <- struct{}{}
  }()

  go func() {
    for nodeName, _ := range sm.Cluster.Nodes {
      if nodeName == sm.State.MyId {
        continue
      }

      client := &http.Client{}
      form := url.Values{}
      form.Set("term", strconv.FormatUint(sm.State.CurrentTerm, 10))
      form.Set("candidateId", sm.State.MyId)
      req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/request-vote", nodeName), bytes.NewBufferString(form.Encode()))
      if err != nil {
        return
      }
      req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
      resp, err := client.Do(req)
      if err != nil {
        return
      }

      switch statusCode := resp.StatusCode; statusCode {
        case 200:
          addOkNode <- struct{}{}
        default:
          return
      }
    }
  }()

  LOOP: for {
    select {
      case <- timer.C:
        timer.Stop()

        // if the majority has been reached, break the loop and become leader
        if uint(okNode) >= sm.Cluster.Majority {
          break LOOP
        }

        return
      case <- done:
        break LOOP
    }
  }

  sm.Timer.Stop()
  sm.State.Switch(LEADER)
  go func() {
    defer func() {
      sm.Timer.Start()
    }()

    for state := sm.State.Is(); state == LEADER; state = sm.State.Is() {
      timer = time.NewTimer(50 * time.Millisecond)

      addOkNode = make(chan struct{}, len(sm.Cluster.Nodes))
      // we start at one because we implicitly count this statemachine
      okNode = 1
      done = make(chan struct{}, 0)

      // counter of node who positivly responded to request-vote
      go func() {
        for ; okNode != len(sm.Cluster.Nodes); okNode++ {
          <- addOkNode
        }

        done <- struct{}{}
      }()

      for nodeName, _ := range sm.Cluster.Nodes {
        if nodeName == sm.State.MyId {
          continue
        }

        go func(nodeName string) {
          defer func() {
            addOkNode <- struct{}{}
          }()

          log.Println("AppendEntry RPC started for: ", sm.State.MyId, " to: ", nodeName)
          client := &http.Client{}
          form := url.Values{}
          form.Set("term", strconv.FormatUint(sm.State.CurrentTerm, 10))
          req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/append-entry?%s", nodeName, form.Encode()), bytes.NewBufferString("[]"))
          if err != nil {
            return
          }
          req.Header.Add("Content-Type", "application/json")
          _, err = client.Do(req)
          if err != nil {
            return
          }
        }(nodeName)
      }

      LOOP: for {
        select {
          case <- timer.C:
            timer.Stop()
            break LOOP
          case <- done:
            <- time.Tick(50 * time.Millisecond)
            break LOOP
        }
      }

    }
  }()
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
  mux.HandleFunc("/append-entry", func(w http.ResponseWriter, r *http.Request) {
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

    //// end parsing prevLogTerm

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

  mux.HandleFunc("/request-vote", func(w http.ResponseWriter, r *http.Request) {
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

    // start parsing candidateId
    candidateId, ok := r.Form["candidateId"]
    if !ok || candidateId[0] == "" {
      statusCode = 422
      return
    }

    candidate := candidateId[0]

    // If the name of the candidate cannot be found in the
    // cluster configuration then tell him to gtfo
    if ok := sm.Cluster. Find(candidate); ok == false {
      statusCode = 422
      return
    }

    // If the candidate's term is the same as the CurrentTerm
    // We need to check if VotedFor is not null AND VotedFor is not the same candidate's ID
    // Otherwise since there is only one VotedFor per term and at start up the VotedFor is null
    // respond to false.
    log.Println(sm.State.MyId, "receive vote: ", r.Form, "votedFor :", sm.State.VotedFor, "myterm :", sm.State.CurrentTerm)
    if newTerm == sm.State.CurrentTerm {
      if sm.State.VotedFor != "" && sm.State.VotedFor != candidate {
        statusCode = 422
        return
      }
    }
    // end parsing candidateId

    // if Voted yes, convert to follower and vote for him
    if sm.State.Is() != FOLLOWER {
      sm.State.Switch(FOLLOWER)
      sm.Timer.Start()
    }

    sm.State.CurrentTerm = newTerm
    sm.State.VotedFor = candidate

  })

  rpc := &RPC{
    Server: server,
    StateMachine: sm,
  }

  return rpc, nil
}
