// Copyright 2014, The Spawn Developers.
// Released under MIT License.

// Package spawn provides multiprocessing functionality. It allows
// distributing tasks over multiple processes.
package spawn

import (
	"errors"
	"os"
	"reflect"
)

// Result channel is used to receive processed tasks.
var Result chan interface{}

// Task channel is used to distribute unprocessed tasks.
var Task chan interface{}

// self stores the input/output pipes of the current process.
// parent stores the input/output pipes of the process' parent. It's only set
// in child processes and contains nil in the dispatcher process.
var (
	self   *actor // pipes of current process, always set.
	parent *actor // pipes of process' parent, only set in children.
)

// The dispatch function is a user defined function that generates tasks and
// writes them to the Task channel.
// The work function is a user defined function that processes tasks and writes
// them to the Result channel.
var (
	dispatch func()
	work     func()
)

// If we are the dispatcher process, ExpandEnv should return an empty string
// because this environment variable should be set for child processes only.
var isDispatcher = os.ExpandEnv("${SPAWN_WORKER}") == ""

func init() {
	if isDispatcher {
		parent = nil
		self = newActor()
	} else {
		parent = newActorFromExtraFiles(3)
		self = newActorFromExtraFiles(7)
	}
}

var taskVal reflect.Value
var taskPtr reflect.Value

// Register records the type of tasks. It accepts a value or a pointer of
// a value for the preferred task type as argument. Only values of that type
// should be sent to the Task and Result channels.
func Register(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return errors.New("nil pointer")
		}
		taskPtr = val
		taskVal = val.Elem()
	} else {
		taskVal = val
	}
	return nil
}

// Dispatch registers the user defined dispatch function. The user defined
// dispatch function gets executed in the parent process. It should send tasks
// to the Task channel and close the Task channel upon exit.
func Dispatch(f func()) {
	dispatch = f
}

// Work registers the user defined work function. The user defined work
// function gets executed in the child processes. It should receive tasks from
// the Tasks channel and send the processed results to the Result channel.
func Work(f func()) {
	work = f
}

// Run creates n child processes and distributes the tasks created by the
// dispatch function among them. It receives the processed tasks and sends
// them to the Result channel.
func Run(n int) error {
	if dispatch == nil {
		return errors.New("no dispatch function")
	}
	if work == nil {
		return errors.New("no work function")
	}
	if !taskVal.IsValid() {
		return errors.New("no task type registered")
	}

	Task = make(chan interface{})
	Result = make(chan interface{})

	if isDispatcher {
		runDispatcher(n)
	} else {
		runWorker()
	}
	return nil
}
