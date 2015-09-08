package raft

import "reflect"

// Callback list    args
// init::done
// state::changed   old, new string

// Javascript events style
// All methods are safe
// Since it is thread safe and only one action
// can take place at the time, adding, executing,
// removing callbacks will may not be in the order
// you want to.
// TODO: keep the global channel for global actions,
// like adding or removing events, but I should add
// a local channel to edit or execute events without
// blocking all the other events.
type Events struct {
  Callbacks map[string][]*Event
  C chan struct{}
}

type Event struct {
  Callback Callback
  Once bool
  Executed bool
}

type Callback func(args ...interface{})

// .On appends a function to an array
// The function Callback should be pass as a reference
// ie: blih := func(args ...interface{}) {}
// sm.State.On("someState", blih)
// sm.State.Off("someState", blih)
// If it is passed like sm.State.On("someState", func(args ...interface{}){})
// the function will never be removable.
func (e *Events) On(name string, cb Callback) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if cbs, found := e.Callbacks[name]; found {
    event := &Event{
      Callback: cb,
      Once: false,
      Executed: false,
    }

    e.Callbacks[name] = append(cbs, event)
  } else {
    e.Callbacks[name] = make([]*Event, 0, 0)

    event := &Event{
      Callback: cb,
      Once: false,
      Executed: false,
    }

    e.Callbacks[name] = append(e.Callbacks[name], event)
  }
}

// .Once appends a function to an array
// When the function is executed, it will be deferenced.
// The function Callback should be pass as a reference
// ie: blih := func(args ...interface{}) {}
// sm.State.Once("someState", blih)
// sm.State.Off("someState", blih)
// If it is passed like sm.State.Once("someState", func(args ...interface{}){})
// the function will never be removable.
func (e *Events) Once(name string, cb Callback) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if cbs, found := e.Callbacks[name]; found {
    event := &Event{
      Callback: cb,
      Once: true,
      Executed: false,
    }

    e.Callbacks[name] = append(cbs, event)
  } else {
    e.Callbacks[name] = make([]*Event, 0, 0)

    event := &Event{
      Callback: cb,
      Once: true,
      Executed: false,
    }

    e.Callbacks[name] = append(e.Callbacks[name], event)
  }
}

// Remove a callback from the array
// The function Callback should be pass as a reference
// ie: blih := func(args ...interface{}) {}
// sm.State.On("someState", blih)
// sm.State.Off("someState", blih)
// If it is passed like sm.State.On("someState", func(args ...interface{}){})
// the function will never be removable.
func (e *Events) Off(name string, cb Callback) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if cbs, found := e.Callbacks[name]; found {
    length := len(cbs)

    foundAt := -1

    for i, event := range cbs{
      if reflect.ValueOf(event.Callback).Pointer() == reflect.ValueOf(cb).Pointer() {
        foundAt = i
        break
      }
    }

    if foundAt == -1 {
      return
    }

    tmp := make([]*Event, length -1)

    index := 0

    for i := 0; i < length; i++ {
      // Compare two pointer value
      if reflect.ValueOf(cbs[i].Callback).Pointer() == reflect.ValueOf(cb).Pointer() {
        continue
      }

      tmp[index] = cbs[i]
      index = index + 1
    }

    e.Callbacks[name] = tmp
  }
}

// Execute async all the callbacks that was registered with .On
func (e *Events) Exec(name string, args ...interface{}) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if cbs, found := e.Callbacks[name]; found {
    length := len(cbs)

    for i := 0; i < length; i++ {
      go func(event *Event) {
        if event.Once == true  && event.Executed == true {
        } else {
          event.Callback(args...)
          event.Executed = true

          if event.Once == true {
            e.Off(name, event.Callback)
          }
        }
      }(cbs[i])
    }
  }
}

// Execute sync all the callbacks that was registered with .On
func (e *Events) ExecSync(name string, args ...interface{}) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if cbs, found := e.Callbacks[name]; found {
    length := len(cbs)

    for i := 0; i < length; i++ {
      event := cbs[i]

      if event.Once == true  && event.Executed == true {
      } else {
        event.Callback(args...)
        event.Executed = true

        if event.Once == true {
          // Unlock, unspool, lock
          // Otherwise we get a deadlock
          // I know that unspooling could take time because some
          // other events could be waiting, but I need to create
          // a lock for each name to avoid global locking.
          <- e.C
          e.Off(name, event.Callback)
          e.C <- struct{}{}
        }
      }
    }
  }
}

// Creates a callback array
func NewEvents() *Events {
  e := &Events{
    Callbacks: make(map[string][]*Event, 0),
    C: make(chan struct{}, 1),
  }

  return e
}
