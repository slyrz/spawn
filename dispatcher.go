package spawn

import (
	"encoding/gob"
	"fmt"
	"os"
	"os/exec"
)

// The childProcess struct combines several objects used to handle a spawned
// child process.
type childProcess struct {
	*actor       // input/output pipe of child process.
	*exec.Cmd    // command to start/stop process.
	*gob.Encoder // encoder to write on the child's input pipe.
	*gob.Decoder // decoder to read from the child's output pipe.
}

// newChildProcess creates a new child process, but doesn't start it yet.
func newChildProcess() *childProcess {
	act := newActor()
	cmd := exec.Command(os.Args[0])

	// Show the output of our child processes on the dispatcher's terminal.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// To make a long story short: Go's multithreaded nature forbids using
	// plain fork. So we have to use fork + exec instead. This starts
	// a completely new process. We use an envirnment variable to tell this
	// process that it is a child, not a dispatcher. To allow communication,
	// we pass the already created pipes as ExtraFiles to the new process.
	// The child process inherits these ExtraFiles as already open files.
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "SPAWN_WORKER=yes")

	// Child receives the same command-line arguments as the dispatcher.
	cmd.Args = os.Args

	// Child inherites these file descriptors as already opened files.
	cmd.ExtraFiles = make([]*os.File, 0, 8)
	cmd.ExtraFiles = append(cmd.ExtraFiles, self.getFiles()...)
	cmd.ExtraFiles = append(cmd.ExtraFiles, act.getFiles()...)

	// Encoder is used to send tasks to child processes.
	// Decoder is used to receive finished tasks from child processes.
	enc := gob.NewEncoder(act.input.writeEnd)
	dec := gob.NewDecoder(act.output.readEnd)

	return &childProcess{act, cmd, enc, dec}
}

// Closes pipes and kill process.
func (p *childProcess) kill() {
	p.actor.close()
	p.Process.Kill()
}

func runDispatcher(n int) {
	pool := make(map[int]*childProcess)
	for i := 0; i < n; i++ {
		proc := newChildProcess()
		proc.Start()
		pool[proc.Process.Pid] = proc
	}

	// tasksSent: number of tasks sent to our childs
	// tasksDone: number of finished tasks returned from our childs
	tasksSent := 0
	tasksDone := 0

	// Idle channel contains the pids of idle child processes.
	Idle := make(chan int, n)

	// Goroutine which reads pids from the process' input pipe.
	// Whenever a child process finishes a task, it writes its pid to our
	// input pipe and the processed task to its own output pipe. We read the
	// pids here, lookup the corresponding child process and read the processed
	// task from the child process' output pipe.
	go func() {
		var pid int

		dec := gob.NewDecoder(self.input.readEnd)
		for {
			err := dec.Decode(&pid)
			if err != nil {
				fmt.Println(err)
				break
			}

			// If our process pool doesn't contain the pid we just read,
			// something went terribly wrong.
			proc, ok := pool[pid]
			if !ok {
				panic("Received unkown Pid.")
			}

			proc.Decoder.DecodeValue(taskVal)
			Result <- taskVal.Interface()
			Idle <- pid
			tasksDone++
		}
		close(Idle)
	}()

	// Goroutine dispatch() sends tasks over the Task channel. We distribute
	// them among our child processes.
	go dispatch()

	// Goroutine which receives tasks from dispatch() and distributes each
	// task to an idle child process. For every received task it waits till it
	// receives a pid on the Idle channel and sends the task to the
	// corresponding child process. After distributing all tasks, it waits
	// for all results and finally kills the child processes.
	go func() {
		for task := range Task {
			pid := <-Idle

			// The pids we receive from Idle should always exist in our
			// process pool
			pool[pid].Encode(task)
			tasksSent++
		}

		// We distributed all tasks. Now we wait until we received all results.
		for tasksDone < tasksSent {
			<-Idle
		}

		// Workers don't terminate on their own. They just block on a read, so
		// we have to kill them now.
		for _, proc := range pool {
			proc.kill()
		}
		close(Result)
	}()

	// Ready, set, go! Start distributing tasks.
	for pid := range pool {
		Idle <- pid
	}
}
