package spawn

import (
	"os"
	"strconv"
)

type pipe struct {
	readEnd  *os.File
	writeEnd *os.File
}

// Create a new pipe using the os.Pipe function.
func newPipe() *pipe {
	pr, pw, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	return &pipe{pr, pw}
}

// Create a pipe from already opened file descriptors.
func newPipeFromExtraFiles(i, j int) *pipe {
	return &pipe{
		os.NewFile(uintptr(i), strconv.Itoa(i)),
		os.NewFile(uintptr(j), strconv.Itoa(j)),
	}
}

func (p *pipe) close() {
	p.readEnd.Close()
	p.writeEnd.Close()
}

type actor struct {
	input  *pipe
	output *pipe
}

func newActor() *actor {
	return &actor{
		newPipe(),
		newPipe(),
	}
}

func newActorFromExtraFiles(off int) *actor {
	return &actor{
		newPipeFromExtraFiles(off, 1+off),
		newPipeFromExtraFiles(2+off, 3+off),
	}
}

func (a *actor) getFiles() []*os.File {
	return []*os.File{
		a.input.readEnd,
		a.input.writeEnd,
		a.output.readEnd,
		a.output.writeEnd,
	}
}

func (a *actor) close() {
	a.input.close()
	a.output.close()
}
