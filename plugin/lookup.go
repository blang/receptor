package plugin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/blang/receptor/pipe"
)

// LookupService looks up watchers and reactors.
type LookupService interface {
	Watcher(name string) (pipe.Watcher, error)
	Reactor(name string) (pipe.Reactor, error)
	Cleanup(timeout time.Duration)
}

var (
	errStrFileNotRegular    = "File %q not a regular file, directory?"
	errStrFileNotExecutable = "File %q is not executable"
)

// Plugin filename patterns
const (
	FileWatcherPrefix = "receptor-watcher-"
	FileReactorPrefix = "receptor-reactor-"
)

// Lookup looks up plugins and manages the setup and teardown phase.
type Lookup struct {
	pluginPath string // Path to lookup plugins
	watchers   map[string]pipe.Watcher
	reactors   map[string]pipe.Reactor
	processes  []*Process
	sockets    []string
}

// NewLookup creates a new Looup instance
func NewLookup(pluginPath string) *Lookup {
	return &Lookup{
		pluginPath: pluginPath,
		watchers:   make(map[string]pipe.Watcher),
		reactors:   make(map[string]pipe.Reactor),
	}
}

// Watcher looks up an watcher identified by name. Manages the startup of the associated plugin-process.
func (s *Lookup) Watcher(name string) (pipe.Watcher, error) {
	if watcher, found := s.watchers[name]; found {
		return watcher, nil
	}
	filePath, err := s.findExecutable(FileWatcherPrefix + name) //TODO: Validate name (no spaces)
	if err != nil {
		return nil, err
	}
	socketPath, err := newSocket()
	if err != nil {
		return nil, err
	}
	s.addSocket(socketPath)
	p := NewProcess(filePath, []string{"unix", socketPath}, name)
	err = p.Start()
	if err != nil {
		return nil, err
	}
	s.addProcess(p)

	watcher, err := NewRPCWatcher(socketPath)
	if err != nil {
		return nil, err
	}
	s.watchers[name] = watcher
	return watcher, nil
}

// Reactor looks up an reactor identified by name. Manages the startup of the associated plugin-process.
func (s *Lookup) Reactor(name string) (pipe.Reactor, error) {
	if reactor, found := s.reactors[name]; found {
		return reactor, nil
	}
	filePath, err := s.findExecutable(FileReactorPrefix + name) //TODO: Validate name (no spaces)
	if err != nil {
		return nil, err
	}
	socketPath, err := newSocket()
	if err != nil {
		return nil, err
	}
	s.addSocket(socketPath)
	p := NewProcess(filePath, []string{"unix", socketPath}, name)
	err = p.Start()
	if err != nil {
		return nil, err
	}
	s.addProcess(p)

	reactor, err := NewRPCReactor(socketPath)
	if err != nil {
		return nil, err
	}
	s.reactors[name] = reactor
	return reactor, nil
}

func (s *Lookup) findExecutable(filename string) (string, error) {
	filePath := filepath.Join(s.pluginPath, filename)
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf(errStrFileNotRegular, filePath)
	}
	if info.Mode()&0111 == 0 { // Check executable flag
		return "", fmt.Errorf(errStrFileNotExecutable, filePath)
	}
	return filePath, nil
}

func (s *Lookup) addSocket(socketPath string) {
	s.sockets = append(s.sockets, socketPath)
}

func (s *Lookup) addProcess(process *Process) {
	s.processes = append(s.processes, process)
}

// Cleanup kills all plugin processes and removes sockets.
// Each cleanup step times out in parallel, which results in a potentional higher timeout for the whole cleanup.
// Blocks until completed.
func (s *Lookup) Cleanup(timeout time.Duration) {
	for _, proc := range s.processes {
		proc.Stop()
	}

	wg := sync.WaitGroup{}
	wg.Add(len(s.processes))
	for _, proc := range s.processes {
		go func(proc *Process) {
			select {
			case <-proc.WaitCh():
			case <-time.After(timeout):
				log.Printf("Process %s timed out while cleanup", proc.actorName)
			}
			wg.Done()
		}(proc)
	}
	wg.Wait()

	for _, socket := range s.sockets {
		err := os.Remove(socket)
		if err != nil {
			log.Printf("Could not remove socket %s", socket)
		}
	}
}
