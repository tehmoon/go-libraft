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
  Log []interface{}

  // Maintain the index in memory
  Index uint64

  // Sync things
  C chan interface{}

  // The state object the Storage can be reused
  State *State
}

// Append a log to the Logs array.
// It can be any value
// Returns the index
func (s *Storage) AppendLog(payload interface{}) (uint64, error) {
  // Cannot store anything until state has not changed
  if s.State.Is() == INIT {
    return s.Index, fmt.Errorf("System is not ready yet.")
  }

  s.C <- payload

  // Append the log to memory
  // and set the index to the length
  // of the log's array.
  // With on-disk array the index will be incremented
  // like s.Index = s.Index + 1 after log has safely
  // been written to storage.
  s.Log = append(s.Log, payload)
  s.Index = uint64(len(s.Log))

  // Simulate the storage, 10 ms correspond to a ssd access
  <- time.Tick(10 * time.Millisecond)

  // Apply
  payload = <- s.C

  return s.Index, nil
}

// Storage will be initialized
func (s *Storage) Start(ok chan struct{}) error {
  // Simulate some init time
  <- time.Tick(3 * time.Second)

  ok <- struct{}{}

  return nil
}

// Initialize the storage
// Here it's just a bounded in-memory array, everything is lost,
// when starting the program, but you can use that function
// to initialize the storage and the index
func NewStorage(state *State) (*Storage, error) {
  storage := &Storage{
    Log: make([]interface{}, 0),
    Index: 0,
    C: make(chan interface{}, 1),
    State: state,
  }

  return storage, nil
}
