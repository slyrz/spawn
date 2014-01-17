package spawn

import (
	"encoding/gob"
	"os"
)

func runWorker() {
	// Goroutine which reads tasks from the process' input pipe and sends
	// them to the Task channel.
	go func() {
		dec := gob.NewDecoder(self.input.readEnd)
		for {
			err := dec.DecodeValue(taskVal)
			if err != nil {
				panic(err)
			}
			Task <- taskVal.Interface()
		}
	}()

	// Goroutine work() processes tasks from Task channel and sends them
	// to the Result channel.
	go work()

	// encPid: encoder to write pids to input pipe of parent.
	// encRes: encoder to write finished tasks to our own output pipe.
	encPid := gob.NewEncoder(parent.input.writeEnd)
	encRes := gob.NewEncoder(self.output.writeEnd)

	pid := os.Getpid()

	for res := range Result {
		var err error

		// Tell our parent that we finished processing the task.
		err = encPid.Encode(pid)
		if err != nil {
			panic(err)
		}

		// Write process task to our own output pipe. The parent process is
		// going to read it from there.
		err = encRes.Encode(res)
		if err != nil {
			panic(err)
		}
	}
}
