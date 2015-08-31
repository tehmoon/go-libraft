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
  Status string

  // Channel to sync everyting
  C chan string

  // On every change call all the registred callbacks
  *Callbacks
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
  s.Status = state
  s.Exec(state)
  <- s.C

  return nil
}

// Defer should be called to make sure that the channel
// is not waiting on any other call.
// Otherwise, missing calls and error returning will cause
// the init loop to be stuck.
type InitFunc func(ok chan struct{}) error

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
// Accepts a callback as argument so you know when init's done.
func (s *StateMachine) Initialize(cb func()) error {
  if s.Initialized == true {
    err := fmt.Errorf("StateMachine is already initialized.")

    return err
  }

  errC := make(chan error)
  ok := make(chan struct{})

  // That loop starts goroutine, it passes
  // a channel that is used to tell the main
  // program it's ok i'm initialized.
  // If the goroutine is done executing, it sends
  // an error interface, it will be catched by the
  // last loop.
  // Many funcs can be blocking it does not matter,
  // since goroutine are used.
  for _, initFunc := range s.Init {
    go func(cb InitFunc) {
      err := cb(ok)
      if err != nil {
        errC <- err
      }

      errC <- nil
    }(initFunc)
  }

  // Sends ok to catch when funcs are ready
  for _, _ = range s.Init {
    <- ok
  }

  // Funcs are initialized
  s.Initialized = true
  if cb != nil {
    go cb()
  }
  s.State.Switch(FOLLOWER)

  // Catch optionnal errors
  // It will block if some functions blocks
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

  // Main core of Raft
  // Maintain the state of the state machine
  state := &State{
    // Initialize status to INIT to prevent Raft of
    // acting before every service has stated
    Status: INIT,

    C: make(chan string, 1),

    Callbacks: NewCallbacks(),
  }

  s.State = state

  s.Init = append(s.Init, func(ok chan struct{}) error {
    var err error

    defer func(err error) {
      if err != nil {
        ok <- struct{}{}
      }
    }(err)

    storage, err := NewStorage(s.State)
    if err != nil {
      return err
    }

    s.Storage = storage

    err = storage.Start(ok)

    return err
  })

  s.Init = append(s.Init, func(ok chan struct{}) error {
    var err error

    defer func(err error) {
      if err != nil {
        ok <- struct{}{}
      }
    }(err)

    rpc, err := NewRPC()
    if err != nil {
      return err
    }

    s.RPC = rpc

    err = s.RPC.Start(ok)
    if err != nil {
      return err
    }

    return nil
  })

  return s
}
