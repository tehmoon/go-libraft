package raft

import (
  "time"
  "math/rand"
)

// Wrapper around time.Timer
type Timer struct {
  // Where the callback will be stored
  Callback func()

  // Store the Timer object
  // Each time Start() will be called, it will
  // create a new Timer.
  // Each time Stop() will be called, it will
  // set timer.Timer to nil to prevent reusing the Timer.
  Timer *time.Timer

  // Tell the Timer he can reset itself, the callback has been executed
  ResetChan chan struct{}

  // Break the for select loop to destroy the goroutine
  StopChan chan struct{}

  // The timer will be randomized base on Min and Max (in milliSecond)
  Min int
  Max int
}

// Basic function that create randomized between two ints
func Random(min, max int) int {
  // TODO: add more entropy ?
  return rand.Intn(max - min) + min
}

// Start a timer with the stored configuration in *Timer
func (t *Timer) Start() bool {
  // If a timer is already stored, do not run
  if t.Timer != nil {
    return false
  }

  sleepFor := time.Duration(Random(t.Min, t.Max)) * time.Millisecond
  t.Timer = time.AfterFunc(sleepFor, t.Callback)

  // Launch the goroutine which will handle the reset of the timer.
  // When the duration of AfterFunc will elapse, the callback will
  // trigger. After the callback will be executed, a struct{}{} will
  // be send to t.ResetChan, which will be catched by the following
  // goroutine. Ultimatly, the timer will reset, and everything will
  // be executed again until a struct{}{} will be sent to StopChan.
  go func() {
    LOOP: for {
      select {
      case <- t.ResetChan:
        sleepFor := time.Duration(Random(t.Min, t.Max)) * time.Millisecond
        t.Timer.Reset(sleepFor)
      case <- t.StopChan:
        break LOOP
      }
    }
  }()

  return true
}

// Stop the timer, the goroutine will be destroyed and
// timer stopped. The timer object will also be destroyed.
// No callback can be executed after Stop().
// If you want to restart the timer you'll have to:
// if stopped := timer.Stop(); stopped {
//   timer.Start()
// }
// To start the timer all over again
func (t *Timer) Stop() bool {
  t.StopChan <- struct{}{}
  stopped := t.Timer.Stop()

  t.Timer = nil

  return stopped
}

// Create a newTimer
func NewTimer(min, max int, cb func()) *Timer {
  timer := &Timer{
    Timer: nil,
    ResetChan: make(chan struct{}),
    StopChan: make(chan struct{}),
    Min: min,
    Max: max,
  }

  timer.Callback = func() {
    cb()
    timer.ResetChan <- struct{}{}
  }

  return timer
}
