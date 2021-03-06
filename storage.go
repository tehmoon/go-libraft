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
  Logs Logs

  // Maintain the index in memory
  Index int64

  // Sync things
  C chan struct{}

  // The state object the Storage can be reused
  State *State
}

type Logs []Log

type Log struct {
  // The term when the index has been stored
  Term uint64 `json:"term"`

  // The actual log
  Payload interface{} `json:"payload"`

  // Litteral without specify names wont work
  _ struct{}
}

// Append a log to the Logs array.
// It can be any value
// Returns the index
func (s *Storage) AppendLog(index int64, log *Log) (int64, error) {
  // Cannot store anything until state has not changed
  if s.State.Is() == INIT {
    return s.Index, fmt.Errorf("System is not ready yet.")
  }

  if len(s.C) != 1 {
    return s.Index, fmt.Errorf("Storage has to be locked before using it.")
  }

  return 0, fmt.Errorf("NYI")

  if index < int64(len(s.Logs)) {
    s.Logs[index] = *log
    s.Index = index
  } else {
    s.Logs = append(s.Logs, *log)
    s.Index = s.Index + 1
  }

  // Simulate the storage, 10 ms
  <- time.Tick(10 * time.Millisecond)

  return s.Index, nil
}

// Storage will be initialized
func (s *Storage) Start() error {
  return nil
}

// Initialize the storage
// Here it's just a bounded in-memory array, everything is lost,
// when starting the program, but you can use that function
// to initialize the storage and the index
func NewStorage(state *State) (*Storage, error) {
  storage := &Storage{
    Logs: make(Logs, 0),
    Index: -1,
    C: make(chan struct{}, 1),
    State: state,
  }

  return storage, nil
}
