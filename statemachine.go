package raft

import (
  "fmt"
  "time"
)

// Global warmup state
var INIT string = "init"

// Internal struct
type State struct {
  // Status of raft:
  // INIT: raft is in warmup state, usually to make sure services are started
  // FOLLOWER: store logs, follow leader, vote for others
  // CANDIDATE: vote for themselves
  // LEADER: send logs, send heartbeats
  Status string // INIT

  // The currentTerm the server is
  CurrentTerm uint64 // 0

  // The candidateId that received vote in currentTerm
  VotedFor string // ""

  // The Id of this server.
  // It is a concatation of RPCExtHost and RPCExtPort
  // since there is only one network triplet per server.
  MyId string // "0.0.0.0:11321"

  // Channel to sync everyting
  C chan struct{}
  SyncTerm chan struct{}

  *Events
}

// Nicer method to get the current state than STATE.Status
func (s *State) Is() string {
  return s.Status
}

// Safely switch from a state to another
// This should only be called by the Raft package
func (s *State) Switch(state string) error {
  s.C <- struct{}{}
  defer func() {
    <- s.C
  }()

  // If the state does not change, return
  if state == s.Is() {
    return nil
  }

  old := s.Is()

  // Raft cannot be at any state other than FOLLOWER when
  // state is at INIT
  if s.Is() == INIT && state != FOLLOWER {
    return fmt.Errorf("State must be follower when status == INIT")
  }


  // Checking correct state has argument
  switch state {
    case FOLLOWER, CANDIDATE, LEADER:
    default:
      return fmt.Errorf("Only supported operations: FOLLOWER, CANDIDATE or LEADER.")
  }

  // Make the switch
  // Call the function in the cases
  switch state {
    case FOLLOWER:
      <- time.Tick(100 * time.Millisecond)
    case CANDIDATE:
      <- time.Tick(300 * time.Millisecond)
    case LEADER:
      <- time.Tick(500 * time.Millisecond)
  }

  // Apply the status if the case bellow has been
  // correctly executed
  s.Status = state
  s.Exec("state::changed", old, state)

  return nil
}

type Node struct {
  Ip string
  Port uint64
}

// Cluster struct
// The node configuration of the server
// can be retreived with: sm.Cluster.Nodes[sm.MyId]
type Cluster struct {
  // Store all the nodes
  Nodes map[string]Node

  // Sync everything
  C chan struct{}

  // Hardcode the majority of the cluster
  Majority uint
}

func (c *Cluster) Find(name string) bool {
  _, found := c.Nodes[name]

  return found
}

// Add a node to a cluster, the name of the node
// is the RPCIp:RPCPort
func (c *Cluster) Add(n Node) (string, bool) {
  c.C <- struct{}{}
  defer func() {
    <- c.C
  }()

  name := fmt.Sprintf("%s:%d", n.Ip, n.Port)

  if _, ok := c.Nodes[name]; ok {
    return name, false
  }

  c.Nodes[name] = n

  // Formula to compute majority
  c.Majority = uint((len(c.Nodes) / 2) + 1)

  return name, true
}

func NewCluster() *Cluster {
  cluster := &Cluster {
    C: make(chan struct{}, 1),
    Nodes: make(map[string]Node),
  }

  return cluster
}

type StateMachine struct {
  Storage *Storage
  RPC *RPC
  State *State
  Initialized bool
  Timer *Timer

  Cluster *Cluster

  // Configuration of the state machine so it can be reused
  // by everyone.
  Configuration *StateMachineConfiguration

  // On every change call all the registred callbacks
  *Events // NewEvents()
}

// Initialize Init functions.
// Each function are executed into go routines.
// If some funcs are blocking Initialize will block,
// otherwise, goroutines are destroyed.
// If any errors are returned, Initialize will return,
// the error and by that, it will cause all goroutines
// to stop acting.
// Accepts a callback as argument so you know when init's done.
func (s *StateMachine) Initialize() error {
  if s.Initialized == true {
    err := fmt.Errorf("StateMachine is already initialized.")

    return err
  }

  errC := make(chan error)

  s.ExecSync("init::start", s, errC)

  // Funcs are initialized
  s.Initialized = true
  s.State.Switch(FOLLOWER)
  s.Timer.Start()
  s.Exec("init::done")

  // TODO: catch ERR
  // Catch optionnal errors
  // It will block if some functions blocks
  //for _, _ = range s.Init {
    //err := <- errC
    //if err != nil {
      //s.Initialized = false
      //return err
    //}
  //}

  return nil
}

type StateMachineConfiguration struct {
  // The host addr to listen to
  RPCHost string // "0.0.0.0"

  // The port number to listen to
  RPCPort uint64 // 11321

  // The host addr to give to clients (in case of NAT)
  RPCExtHost string // NYI

  // The port number to give to clients (in case of NAT)
  RPCExtPort uint64 // NYI
}

// Create a state machine object that stores every
// component of Raft, use this object to start Raft.
// You also can overide fields before initialize the
// state machine.
// The functions of the state machine are stored in
// an array, and it's synchronized with channels to
// pass references.
func NewStateMachine(config *StateMachineConfiguration) (*StateMachine, error) {
  s := &StateMachine{
    Events: NewEvents(),
  }

  s.Initialized = false

  if config == nil {
    config = &StateMachineConfiguration{}
  }

  if config.RPCHost == "" {
    config.RPCHost = "0.0.0.0"
  }

  if config.RPCPort == 0 {
    config.RPCPort = 11321
  }

  s.Configuration = config

  s.Cluster = NewCluster()
  id, _ := s.Cluster.Add(Node{config.RPCHost, config.RPCPort})

  // Main core of Raft
  // Maintain the state of the state machine
  state := &State{
    // Initialize status to INIT to prevent Raft of
    // acting before every service has stated
    Status: INIT,

    CurrentTerm: 0,
    VotedFor: "",
    MyId: id,
    C: make(chan struct{}, 1),
    SyncTerm: make(chan struct{}, 1),
    Events: s.Events,
  }

  s.State = state

  s.Timer = NewTimer(150, 2000, func() {
    s.ExecSync("timeout::elapsed", s)
  })

  s.Once("init::start", func(args ...interface{}) {
    var err error

    s := args[0].(*StateMachine)
    errC := args[1].(chan error)

    storage, err := NewStorage(s.State)
    if err != nil {
      errC <- err
      return
    }

    s.Storage = storage

    err = storage.Start()
    if err != nil {
      errC <- err
      return
    }
  })

  s.Once("init::start", func(args ...interface{}) {
    var err error

    s := args[0].(*StateMachine)
    errC := args[1].(chan error)

    rpc, err := NewRPC(s)
    if err != nil {
      errC <- err
      return
    }

    s.RPC = rpc

    go func() {
      err = s.RPC.Start()
      if err != nil {
        errC <- err
        return
      }
    }()
  })

  s.On("timeout::elapsed", func(args ...interface{}) {
    s := args[0].(*StateMachine)

    fmt.Println("timeout elapsed!")
    if s.State.Is() == FOLLOWER {
      s.State.Switch(CANDIDATE)
      s.RPC.StartElection()
    }
    if s.State.Is() == CANDIDATE {
      s.RPC.StartElection()
    }
  })

  return s, nil
}
