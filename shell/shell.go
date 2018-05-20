package shell

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/cortesi/termlog"
	"github.com/google/shlex"
)

var Default = "bash"

type Executor interface {
	Name() string
	Run(command string, log termlog.Stream, bufferr bool) (error, error, string)
	Stop() error
}

var shells = make(map[string]Executor)

func init() {
	register(&Exec{})
	register(&Bash{})
	register(&Builtin{})
}

// Register a new shell interface.
func register(i Executor) {
	name := i.Name()
	if _, has := shells[name]; has {
		panic("shell interface " + name + " already exists")
	}
	shells[name] = i
}

/* execRunner runs a command through exec, streaming output to logs. If bufferr
is true, errors are buffered up and returned. Otherwise, the return string is
always empty. */
func execRunner(c *exec.Cmd, log termlog.Stream, bufferr bool) (
	error, // Invocation error
	error, // Process exit error
	string, // Error buffer
) {
	stdo, err := c.StdoutPipe()
	if err != nil {
		return err, nil, ""
	}
	stde, err := c.StderrPipe()
	if err != nil {
		return err, nil, ""
	}
	buff := new(bytes.Buffer)
	mut := sync.Mutex{}
	err = c.Start()
	if err != nil {
		return err, nil, ""
	}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go logOutput(
		&wg, stde,
		func(s string, args ...interface{}) {
			log.Warn(s, args...)
			if bufferr {
				mut.Lock()
				defer mut.Unlock()
				fmt.Fprintf(buff, "%s\n", args...)
			}
		},
	)
	go logOutput(&wg, stdo, log.Say)
	wg.Wait()
	return nil, c.Wait(), buff.String()
}

func logOutput(wg *sync.WaitGroup, fp io.ReadCloser, out func(string, ...interface{})) {
	defer wg.Done()
	r := bufio.NewReader(fp)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return
		}
		out("%s", string(line))
	}
}

// GetExecutor retrieves an executor, and returns nil if it doesn't exist.
func GetExecutor(method string) Executor {
	exec, has := shells[method]
	if !has {
		return nil
	}
	return exec
}

// No shell, just execute the command raw.
type Exec struct{}

func (r *Exec) Name() string {
	return "exec"
}

func (r *Exec) Run(line string, log termlog.Stream, bufferr bool) (error, error, string) {
	ss, err := shlex.Split(line)
	if err != nil {
		return err, nil, ""
	}
	if len(ss) == 0 {
		return errors.New("No command defined"), nil, ""
	}
	return execRunner(exec.Command(ss[0], ss[1:]...), log, bufferr)
}

func (r *Exec) Stop() error {
	return nil
}

// Bash shell command.
type Bash struct{}

func (b *Bash) Name() string {
	return "bash"
}

func (b *Bash) getShell() (string, error) {
	if _, err := exec.LookPath("bash"); err == nil {
		return "bash", nil
	}
	if _, err := exec.LookPath("sh"); err == nil {
		return "sh", nil
	}
	return "", fmt.Errorf("Could not find bash or sh on path.")
}

func (b *Bash) Run(line string, log termlog.Stream, bufferr bool) (error, error, string) {
	sh, err := b.getShell()
	if err != nil {
		return err, nil, ""
	}
	return execRunner(exec.Command(sh, "-c", line), log, bufferr)
}

func (r *Bash) Stop() error {
	return nil
}

// Builtin shell command.
type Builtin struct{}

func (b *Builtin) Name() string {
	return "builtin"
}

func (b *Builtin) Run(line string, log termlog.Stream, bufferr bool) (error, error, string) {
	path, err := os.Executable()
	if err != nil {
		return err, nil, ""
	}
	return execRunner(exec.Command(path, "--exec", line), log, bufferr)
}

func (r *Builtin) Stop() error {
	return nil
}
