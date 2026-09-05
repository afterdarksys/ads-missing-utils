package runreceipt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestProcessHelper(t *testing.T) {
	if os.Getenv("RECEIPT_HELPER") != "1" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "fail":
		os.Stdout.WriteString("evidence")
		os.Exit(7)
	case "copy":
		io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	case "sleep":
		time.Sleep(time.Minute)
		os.Exit(0)
	}
}
func helper(t *testing.T, mode string) []string {
	t.Helper()
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	return []string{exe, "-test.run=^TestProcessHelper$", "--", mode}
}
func TestPipelinePreservesEarlierFailure(t *testing.T) {
	t.Setenv("RECEIPT_HELPER", "1")
	var out bytes.Buffer
	calls := 0
	r, e := Execute(context.Background(), [][]string{helper(t, "fail"), helper(t, "copy")}, "test", t.TempDir(), nil, &out, io.Discard, func(r Receipt) error { calls++; return nil })
	if e != nil || calls != 2 || ExitCode(r) != 1 || *r.Stages[0].ExitCode != 7 || *r.Stages[1].ExitCode != 0 || out.String() != "evidence" || r.OutcomeVerified {
		t.Fatalf("receipt=%+v error=%v output=%q", r, e, out.String())
	}
}
func TestTimeoutAndMissingExecutable(t *testing.T) {
	t.Setenv("RECEIPT_HELPER", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	r, e := Execute(ctx, [][]string{helper(t, "sleep")}, "test", t.TempDir(), nil, io.Discard, io.Discard, func(Receipt) error { return nil })
	if e != nil || ExitCode(r) != 124 || r.State != "incomplete" {
		t.Fatalf("%+v %v", r, e)
	}
	r, e = Execute(context.Background(), [][]string{{"/nonexistent/receipt-test"}}, "test", t.TempDir(), nil, io.Discard, io.Discard, func(Receipt) error { return nil })
	if e != nil || r.Reason != "stage_launch_failed" || ExitCode(r) != 4 {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestPersistenceFailureAndPrivacy(t *testing.T) {
	t.Setenv("RECEIPT_HELPER", "1")
	var out bytes.Buffer
	r, e := Execute(context.Background(), [][]string{helper(t, "fail")}, "test", t.TempDir(), nil, &out, io.Discard, func(Receipt) error { return errors.New("disk full") })
	if e == nil || out.Len() != 0 || r.Stages[0].State != "not_started" {
		t.Fatal("execution without durable intent")
	}
	p, e := Preview([][]string{{"echo", "secret-value"}}, "test", ".")
	b, _ := json.Marshal(p)
	if e != nil || bytes.Contains(b, []byte("secret-value")) || p.State != "preview" {
		t.Fatal(string(b))
	}
}
