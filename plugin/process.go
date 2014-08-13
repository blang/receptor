package plugin

import (
	"bufio"
	"log"
	"os/exec"
)

// Process manages the start and shutdown of a plugin process
type Process struct {
	path      string
	args      []string
	doneCh    chan struct{}
	pcmd      *exec.Cmd
	actorName string // Name of watcher/reactor used for logs
}

func NewProcess(path string, args []string, actorName string) *Process {
	return &Process{
		path:      path,
		args:      args,
		doneCh:    make(chan struct{}),
		actorName: actorName,
	}
}

func (p *Process) Start() error {
	p.pcmd = exec.Command(p.path, p.args...)
	stdout, err := p.pcmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := p.pcmd.StderrPipe()
	if err != nil {
		return err
	}

	p.pcmd.Start()
	// Wait for plugin to print socket information, signals plugin is ready
	rd := bufio.NewReader(stdout)
	_, err = rd.ReadString('\n')
	if err != nil {
		p.Stop()
		return err
	}
	go func() {
		for {
			line, err := rd.ReadString('\n')
			if err != nil {
				return
			}
			log.Printf("[Plugin %s] %s", p.actorName, line)
		}
	}()
	go func() {
		errRd := bufio.NewReader(stderr)
		for {
			line, err := errRd.ReadString('\n')
			if err != nil {
				return
			}
			log.Printf("[Plugin %s] %s", p.actorName, line)
		}
	}()
	go func() {
		p.pcmd.Wait()
		close(p.doneCh)
		log.Printf("[Plugin %s] Process stopped", p.actorName)
	}()
	return nil
}

func (p *Process) Wait() {
	<-p.doneCh
}

func (p *Process) WaitCh() <-chan struct{} {
	return p.doneCh
}

func (p *Process) Stop() {
	p.pcmd.Process.Kill()
}
