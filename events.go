package raft

import "reflect"

// Callback list    args
// init::done
// state::changed   old, new string
// term::changed    old, new int64

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
  Maps map[string][]*Event
  //Callbacks map[string]*struct{
    //C chan struct{}

  //}
  C chan struct{}
}

type Event struct {
  CallbackFunc CallbackFunc
  Once bool
  Executed bool
}

type CallbackFunc func(args ...interface{})

// .On appends a function to an array
// The function Callback should be pass as a reference
// ie: blih := func(args ...interface{}) {}
// sm.State.On("someState", blih)
// sm.State.Off("someState", blih)
// If it is passed like sm.State.On("someState", func(args ...interface{}){})
// the function will never be removable.
func (e *Events) On(name string, cb CallbackFunc) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if cbs, found := e.Maps[name]; found {
    event := &Event{
      CallbackFunc: cb,
      Once: false,
      Executed: false,
    }

    e.Maps[name] = append(cbs, event)
  } else {
    e.Maps[name] = make([]*Event, 0, 0)

    event := &Event{
      CallbackFunc: cb,
      Once: false,
      Executed: false,
    }

    e.Maps[name] = append(e.Maps[name], event)
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
func (e *Events) Once(name string, cb CallbackFunc) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if cbs, found := e.Maps[name]; found {
    event := &Event{
      CallbackFunc: cb,
      Once: true,
      Executed: false,
    }

    e.Maps[name] = append(cbs, event)
  } else {
    e.Maps[name] = make([]*Event, 0, 0)

    event := &Event{
      CallbackFunc: cb,
      Once: true,
      Executed: false,
    }

    e.Maps[name] = append(e.Maps[name], event)
  }
}

// Remove a callback from the array
// The function Callback should be pass as a reference
// ie: blih := func(args ...interface{}) {}
// sm.State.On("someState", blih)
// sm.State.Off("someState", blih)
// If it is passed like sm.State.On("someState", func(args ...interface{}){})
// the function will never be removable.
func (e *Events) Off(name string, cb CallbackFunc) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if cbs, found := e.Maps[name]; found {
    length := len(cbs)

    foundAt := -1

    for i, event := range cbs{
      if reflect.ValueOf(event.CallbackFunc).Pointer() == reflect.ValueOf(cb).Pointer() {
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
      if reflect.ValueOf(cbs[i].CallbackFunc).Pointer() == reflect.ValueOf(cb).Pointer() {
        continue
      }

      tmp[index] = cbs[i]
      index = index + 1
    }

    e.Maps[name] = tmp
  }
}

// Execute async all the callbacks that was registered with .On
func (e *Events) Exec(name string, args ...interface{}) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if cbs, found := e.Maps[name]; found {
    length := len(cbs)

    for i := 0; i < length; i++ {
      go func(event *Event) {
        if event.Once == true  && event.Executed == true {
        } else {
          event.CallbackFunc(args...)
          event.Executed = true

          if event.Once == true {
            e.Off(name, event.CallbackFunc)
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

  if cbs, found := e.Maps[name]; found {
    length := len(cbs)

    for i := 0; i < length; i++ {
      event := cbs[i]

      if event.Once == true  && event.Executed == true {
      } else {
        event.CallbackFunc(args...)
        event.Executed = true

        if event.Once == true {
          // Unlock, unspool, lock
          // Otherwise we get a deadlock
          // I know that unspooling could take time because some
          // other events could be waiting, but I need to create
          // a lock for each name to avoid global locking.
          <- e.C
          e.Off(name, event.CallbackFunc)
          e.C <- struct{}{}
        }
      }
    }
  }
}

// Creates a callback array
func NewEvents() *Events {
  e := &Events{
    Maps: make(map[string][]*Event, 0),
    C: make(chan struct{}, 1),
  }

  return e
}
