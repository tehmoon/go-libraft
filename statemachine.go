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
}

// Safely switch from a state to another
// This should only be called by the Raft package
func (s *State) Switch(state string) error {
  // If the state does not change, return
  if state == STATE.Status {
    return nil
  }

  // Raft cannot be at any state other than FOLLOWER when
  // state is at INIT
  if STATE.Status == INIT && state != FOLLOWER {
    return fmt.Errorf("State must be follower when STATE == INIT")
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

type StateMachine struct {
  Init []func()
}

func (s *StateMachine) Initialize() error {
  return nil
}

func NewStateMachine() {
}
