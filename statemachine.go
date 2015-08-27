package raft

import (
  "fmt"
  "time"
  "log"
)

var INIT string = "init"

var STATE *State = &State{
  Status: INIT,
  C: make(chan string, 1),
}

type State struct {
  Status string
  C chan string
}

func (s *State) Switch(state string) error {
  if state == STATE.Status {
    return nil
  }

  if STATE.Status == INIT && state != FOLLOWER {
    return fmt.Errorf("State must be follower when STATE == INIT")
  }

  switch state {
    case FOLLOWER, CANDIDATE, LEADER:
      s.C <- state
    default:
      return fmt.Errorf("Only supported operations: FOLLOWER, CANDIDATE or LEADER.")
  }

  switch state {
    case FOLLOWER:
      <- time.Tick(100 * time.Millisecond)
    case CANDIDATE:
      <- time.Tick(300 * time.Millisecond)
    case LEADER:
      <- time.Tick(500 * time.Millisecond)
  }

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
