package raft

import (
  "fmt"
  "time"
)

type Storage struct {
  Log []interface{}
  Index uint64
  C chan interface{}
}

func (s *Storage) AppendLog(payload interface{}) (uint64, error) {
  if STATE.Status == INIT {
    return s.Index, fmt.Errorf("System is not ready yet.")
  }

  s.C <- payload

  s.Log = append(s.Log, payload)
  s.Index = uint64(len(s.Log))

  <- time.Tick(10 * time.Millisecond)

  payload = <- s.C

  return s.Index, nil
}

func NewStorage() (*Storage, error) {
  storage := &Storage{
    Log: make([]interface{}, 0),
    Index: 0,
    C: make(chan interface{}, 1),
  }

  return storage, nil
}
