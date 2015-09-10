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
// like adding or removing callbacks, but I should add
// a local channel to edit or execute callbacks without
// blocking all the other callbacks.
type Events struct {
  //Names map[string][]*Callback
  Names map[string]*Event

  C chan struct{}
}

type Event struct{
  C chan struct{}
  Callbacks []*Callback
}

type Callback struct {
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

  if event, found := e.Names[name]; found {
    callback := &Callback{
      CallbackFunc: cb,
      Once: false,
      Executed: false,
    }

    event.Callbacks = append(event.Callbacks, callback)
  } else {
    event := &Event{
      Callbacks: make([]*Callback, 0, 0),
      C: make(chan struct{}, 1),
    }

    callback := &Callback{
      CallbackFunc: cb,
      Once: false,
      Executed: false,
    }

    event.Callbacks = append(event.Callbacks, callback)

    e.Names[name] = event
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

  if event, found := e.Names[name]; found {
    callback := &Callback{
      CallbackFunc: cb,
      Once: true,
      Executed: false,
    }

    event.Callbacks = append(event.Callbacks, callback)
  } else {
    event := &Event{
      Callbacks: make([]*Callback, 0, 0),
      C: make(chan struct{}, 1),
    }

    callback := &Callback{
      CallbackFunc: cb,
      Once: true,
      Executed: false,
    }

    event.Callbacks = append(event.Callbacks, callback)

    e.Names[name] = event
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

  if event, found := e.Names[name]; found {
    length := len(event.Callbacks)

    foundAt := -1

    for i, callback := range event.Callbacks {
      if reflect.ValueOf(callback.CallbackFunc).Pointer() == reflect.ValueOf(cb).Pointer() {
        foundAt = i
        break
      }
    }

    if foundAt == -1 {
      return
    }

    tmp := make([]*Callback, length -1)

    index := 0

    for i := 0; i < length; i++ {
      // Compare two pointer value
      if reflect.ValueOf(event.Callbacks[i].CallbackFunc).Pointer() == reflect.ValueOf(cb).Pointer() {
        continue
      }

      tmp[index] = event.Callbacks[i]
      index = index + 1
    }

    e.Names[name].Callbacks = tmp
  }
}

// Execute async all the callbacks that was registered with .On
func (e *Events) Exec(name string, args ...interface{}) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if event, found := e.Names[name]; found {
    length := len(event.Callbacks)

    for i := 0; i < length; i++ {
      go func(callback *Callback) {
        if callback.Once == true  && callback.Executed == true {
        } else {
          callback.CallbackFunc(args...)
          callback.Executed = true

          if callback.Once == true {
            // Async delete of the unused CallbackFun
            go e.Off(name, callback.CallbackFunc)
          }
        }
      }(event.Callbacks[i])
    }
  }
}

// Execute sync all the callbacks that was registered with .On
func (e *Events) ExecSync(name string, args ...interface{}) {
  e.C <- struct{}{}
  defer func() {
    <- e.C
  }()

  if event, found := e.Names[name]; found {
    length := len(event.Callbacks)

    for i := 0; i < length; i++ {
      callback := event.Callbacks[i]

      if callback.Once == true  && callback.Executed == true {
      } else {
        callback.CallbackFunc(args...)
        callback.Executed = true

        if callback.Once == true {
          // Async delete of the unused CallbackFun
          go e.Off(name, callback.CallbackFunc)
        }
      }
    }
  }
}

// Creates a callback array
func NewEvents() *Events {
  e := &Events{
    Names: make(map[string]*Event, 0),
    C: make(chan struct{}, 1),
  }

  return e
}
