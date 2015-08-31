package raft

import "reflect"

// Javascript register callback style
// All methods are safe
type Callbacks struct {
  Callbacks []Callback
  C chan struct{}
}

type Callback func(state string)

// .On appends a function to an array
// The function Callback should be pass as a reference
// ie: blih := func(state string) {}
// sm.State.On(blih)
// sm.State.Remove(blih)
// If it is passed like sm.State.On(func(s string){})
// the function will never be removable.
func (c *Callbacks) On(cb Callback) {
  c.C <- struct{}{}

  c.Callbacks = append(c.Callbacks, cb)

  <- c.C
}

// Remove a callback from the array
// The function Callback should be pass as a reference
// ie: blih := func(state string) {}
// sm.State.On(blih)
// sm.State.Remove(blih)
// If it is passed like sm.State.On(func(s string){})
// the function will never be removable.
func (c *Callbacks) Remove(cb Callback) {
  c.C <- struct{}{}
  defer func() {
    <- c.C
  }()

  length := len(c.Callbacks)

  foundAt := -1

  for i, f := range c.Callbacks {
    if reflect.ValueOf(f).Pointer() == reflect.ValueOf(cb).Pointer() {
      foundAt = i
      break
    }
  }

  if foundAt == -1 {
    return
  }

  tmp := make([]Callback, length -1)

  index := 0

  for i := 0; i < length; i++ {
    if reflect.ValueOf(c.Callbacks[i]).Pointer() == reflect.ValueOf(cb).Pointer() {
      continue
    }

    tmp[index] = c.Callbacks[i]
    index = index + 1
  }


  c.Callbacks = tmp
}

// Execute all the callbacks that was registered with .On
func (c *Callbacks) Exec(state string) {
  c.C <- struct{}{}

  length := len(c.Callbacks)

  synchronize := make(chan struct{}, 0)

  for i := 0; i < length; i++ {
    go func(f Callback) {
      f(state)
      synchronize <- struct{}{}
    }(c.Callbacks[i])
  }

  for i := 0; i < length; i++ {
    <- synchronize
  }

  <- c.C
}

// Creates a callback array
func NewCallbacks() *Callbacks {
  cb := &Callbacks{
    Callbacks: make([]Callback, 0, 0),
    C: make(chan struct{}, 1),
  }

  return cb
}
