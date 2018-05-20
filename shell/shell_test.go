package shell

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/cortesi/termlog"
)

type cmdTest struct {
	cmd     string
	bufferr bool

	logHas  string
	buffHas string
	err     bool
	procerr bool
}

func testCmd(t *testing.T, ex Executor, ct cmdTest) {
	lt := termlog.NewLogTest()
	err, procerr, buff := ex.Run(ct.cmd, lt.Log.Stream(""), ct.bufferr)
	if (err != nil) != ct.err {
		t.Errorf("Unexpected invocation error: %s", err)
	}
	if (procerr != nil) != ct.procerr {
		t.Errorf("Unexpected process error: %s", err)
	}
	if ct.buffHas != "" && !strings.Contains(buff, ct.buffHas) {
		t.Errorf("Unexpected buffer return: %s", buff)
	}
	if ct.logHas != "" && !strings.Contains(lt.String(), ct.logHas) {
		t.Errorf("Unexpected log return: %s", lt.String())
	}
}

var bashTests = []cmdTest{
	{
		cmd:    "echo moddtest; true",
		logHas: "moddtest",
	},
	{
		cmd:     "echo moddtest; false",
		logHas:  "moddtest",
		procerr: true,
	},
	{
		cmd:     "definitelynosuchcommand",
		procerr: true,
	},
	{
		cmd:     "echo moddstderr >&2",
		bufferr: true,
		buffHas: "moddstderr",
	},
}

func TestBash(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("skipping bash test")
		return
	}
	b := Bash{}
	for _, tc := range bashTests {
		testCmd(t, &b, tc)
	}
}
