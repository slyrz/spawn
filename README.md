# spawn

*Process-based parallelism for the Go programming language*

The spawn package provides multiprocessing funcionality. It
allows spawning processes and distributing tasks among them.

## Quickstart

The following example shows how to calculate the squares of the
first 1000 numbers across 4 processes.

```go
package main

import (
  "fmt"
  "github.com/slyrz/spawn"
)

func main() {
  var err error

  // We are going to pass ints as tasks in this example,
  // so we have to register type int here.
  err = spawn.Register(new(int))
  if err != nil {
    panic(err)
  }

  // The dispatch function generates tasks and submits them to
  // the worker processes.
  spawn.Dispatch(func() {
    for i := 1; i <= 1000; i++ {
      spawn.Task <- i
    }
    close(spawn.Task)
  })

  // Each worker process receives tasks, performs a heavy computation (x^2)
  // and sends the processed tasks back to our main process.
  spawn.Work(func() {
    for task := range spawn.Task {
      val := task.(int)
      spawn.Result <- (val * val)
    }
  })

  // Let's spawn 4 worker processes.
  err = spawn.Run(4)
  if err != nil {
    panic(err)
  }

  // Receive all results in a semi-random order.
  for res := range spawn.Result {
    fmt.Println(res)
  }
}
```

### License

spawn is released under MIT license.
You can find a copy of the MIT License in the [LICENSE](./LICENSE) file.

