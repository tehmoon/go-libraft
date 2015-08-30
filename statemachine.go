package raft

import (
  "fmt"
  "time"
)

// Global warmup state
var INIT string = "init"

// Global State Object, main core of Raft
var STATE *State = &State{
  // Initialize status to INIT to prevent Raft of
  // acting before every service has stated
  Status: INIT,

  C: make(chan string, 1),
}

type Callbacks struct {
  Callbacks []Callback
  C chan Callback
}

type Callback func(state string)

func (c *Callbacks) On(cb Callback) {
  c.C <- cb

  c.Callbacks = append(c.Callbacks, <- c.C)
}

// Internal struct
type State struct {
  // Status of raft:
  // INIT: raft is in warmup state, usually to make sure services are started
  // FOLLOWER: store logs, follow leader, vote for others
  // CANDIDATE: vote for themselves
  // LEADER: send logs, send heartbeats
  Status string

  // Channel to sync everyting
  C chan string

  // On every change call all the registred callbacks
  Callbacks
}

// Nicer method to get the current state than STATE.Status
func (s *State) Is() string {
  return s.Status
}

// Safely switch from a state to another
// This should only be called by the Raft package
func (s *State) Switch(state string) error {
  // If the state does not change, return
  if state == s.Is() {
    return nil
  }

  // Raft cannot be at any state other than FOLLOWER when
  // state is at INIT
  if s.Is() == INIT && state != FOLLOWER {
    return fmt.Errorf("State must be follower when status == INIT")
  }

  // Checking correct state has argument
  switch state {
    case FOLLOWER, CANDIDATE, LEADER:
      s.C <- state
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
  s.Status = <- s.C

  return nil
}

type InitFunc func() error

type StateMachine struct {
  Init []InitFunc
  Storage *Storage
  RPC *RPC
  State *State
  Initialized bool
}

// Initialize Init functions.
// Each function are executed into go routines.
// If some funcs are blocking Initialize will block,
// otherwise, goroutines are destroyed.
// If any errors are returned, Initialize will return,
// the error and by that, it will cause all goroutines
// to stop acting.
func (s *StateMachine) Initialize() error {
  if s.Initialized == true {
    err := fmt.Errorf("StateMachine is already initialized.")

    return err
  }

  errC := make(chan error)

  for _, initFunc := range s.Init {
    go func(cb InitFunc) {
      err := cb()
      if err != nil {
        errC <- err
      }

      errC <- nil
    }(initFunc)
  }

  s.Initialized = true

  for _, _ = range s.Init {
    err := <- errC
    if err != nil {
      s.Initialized = false
      return err
    }
  }

  return nil
}

// Create a state machine object that stores every
// component of Raft, use this object to start Raft.
// You also can overide fields before initialize the
// state machine.
// The functions of the state machine are stored in
// an array, and it's synchronized with channels to
// pass references.
func NewStateMachine() *StateMachine  {
  s := &StateMachine{
    Init: make([]InitFunc, 0, 0),
  }

  s.Initialized = false
  s.State = STATE

  s.Init = append(s.Init, func() error {
    storage, err := NewStorage()
    s.Storage = storage

    return err
  })

  s.Init = append(s.Init, func() error {
    rpc, err := NewRPC()
    if err != nil {
      return err
    }

    s.RPC = rpc

    err = s.RPC.Start()
    if err != nil {
      return err
    }

    return nil
  })

  return s
}
