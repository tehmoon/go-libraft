package raft

import "reflect"

// Callback list    args
// init::done
//

// Javascript register callback style
// All methods are safe
type Callbacks struct {
  Callbacks map[string][]Callback
  C chan struct{}
}

type Callback func(args ...interface{})

// .On appends a function to an array
// The function Callback should be pass as a reference
// ie: blih := func(args ...interface{}) {}
// sm.State.On("someState", blih)
// sm.State.Off("someState", blih)
// If it is passed like sm.State.On("someState", func(args ...interface{}){})
// the function will never be removable.
func (c *Callbacks) On(name string, cb Callback) {
  c.C <- struct{}{}

  if cbs, found := c.Callbacks[name]; found {
    c.Callbacks[name] = append(cbs, cb)
  } else {
    c.Callbacks[name] = make([]Callback, 0, 0)
    c.Callbacks[name] = append(c.Callbacks[name], cb)
  }

  <- c.C
}

// Remove a callback from the array
// The function Callback should be pass as a reference
// ie: blih := func(args ...interface{}) {}
// sm.State.On("someState", blih)
// sm.State.Off("someState", blih)
// If it is passed like sm.State.On("someState", func(args ...interface{}){})
// the function will never be removable.
func (c *Callbacks) Off(name string, cb Callback) {
  c.C <- struct{}{}
  defer func() {
    <- c.C
  }()

  if cbs, found := c.Callbacks[name]; found {
    length := len(cbs)

    foundAt := -1

    for i, f := range cbs {
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
      // Compare two pointer value
      if reflect.ValueOf(cbs[i]).Pointer() == reflect.ValueOf(cb).Pointer() {
        continue
      }

      tmp[index] = cbs[i]
      index = index + 1
    }

    c.Callbacks[name] = tmp
  }
}

// Execute all the callbacks that was registered with .On
func (c *Callbacks) Exec(name string, args ...interface{}) {
  c.C <- struct{}{}

  defer func() {
    <- c.C
  }()

  if cbs, found := c.Callbacks[name]; found {
    length := len(cbs)

    for i := 0; i < length; i++ {
      go func(f Callback) {
        f(args)
      }(cbs[i])
    }
  }
}

// Creates a callback array
func NewCallbacks() *Callbacks {
  cb := &Callbacks{
    Callbacks: make(map[string][]Callback, 0),
    C: make(chan struct{}, 1),
  }

  return cb
}
