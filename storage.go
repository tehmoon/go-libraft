package raft

import (
  "fmt"
  "time"
)

// Used to store logs.
// This must not be modified by someone execpt Storage's
// methods.
type Storage struct {
  // Logs are in memory for now but it's going to change
  // when Raft will be implemented
  Logs []*Log

  // Maintain the index in memory
  Index uint64

  // Sync things
  C chan struct{}

  // The state object the Storage can be reused
  State *State
}

type Log struct {
  // The term when the index has been stored
  Term uint64

  // The actual log
  Payload interface{}

  // Litteral without specify names wont work
  _ struct{}
}

// Append a log to the Logs array.
// It can be any value
// Returns the index
func (s *Storage) AppendLog(term uint64, payload interface{}) (uint64, error) {
  // Cannot store anything until state has not changed
  if s.State.Is() == INIT {
    return s.Index, fmt.Errorf("System is not ready yet.")
  }

  s.C <- struct{}{}

  // Append the log to memory
  // and set the index to the length
  // of the log's array.
  // With on-disk array the index will be incremented
  // like s.Index = s.Index + 1 after log has safely
  // been written to storage.
  log := &Log{
    Term: term,
    Payload: payload,
  }

  s.Logs = append(s.Logs, log)
  s.Index = s.Index + 1

  // Simulate the storage, 10 ms
  <- time.Tick(10 * time.Millisecond)

  <- s.C

  return s.Index, nil
}

// Storage will be initialized
func (s *Storage) Start() error {
  // Simulate some init time
  <- time.Tick(5 * time.Second)

  return nil
}

// Initialize the storage
// Here it's just a bounded in-memory array, everything is lost,
// when starting the program, but you can use that function
// to initialize the storage and the index
func NewStorage(state *State) (*Storage, error) {
  storage := &Storage{
    Logs: make([]*Log, 0),
    Index: 0,
    C: make(chan struct{}, 1),
    State: state,
  }

  return storage, nil
}
