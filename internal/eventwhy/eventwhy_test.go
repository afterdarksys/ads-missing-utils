package eventwhy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func event(id, action string, at int) string {
	return fmt.Sprintf(`{"Type":"container","Action":%q,"time":%d,"Actor":{"ID":%q,"Attributes":{"exitCode":"137","secret":"never-export","name":"private-label"}}}`, action, at, id)
}
func TestGroupsRepeatedEventsWithoutRootCauseGuess(t *testing.T) {
	data := event("a", "die", 100) + "\n" + event("a", "die", 102) + "\n" + event("b", "die", 103)
	r := Normalize(strings.NewReader(data), "provided", "demo", 20)
	if r.Outcome != "pass" || r.Events != 3 || len(r.Groups) != 2 || r.Groups[0].Count != 2 || r.Groups[0].First.Unix() != 100 || r.Groups[0].Last.Unix() != 102 {
		t.Fatalf("%+v", r)
	}
	raw, _ := json.Marshal(r)
	for _, secret := range []string{"never-export", "private-label"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("unallowlisted metadata retained")
		}
	}
}
func TestEventGapsStayPartial(t *testing.T) {
	for _, input := range []string{`{}`, `not-json`, event("a", "die", 100) + "\n" + event("a", "die", 101), strings.Repeat("x", 300<<10)} {
		r := Normalize(strings.NewReader(input), "provided", "demo", 1)
		if r.Outcome != "partial" || len(r.Diagnostics) == 0 {
			t.Fatal("gap hidden")
		}
	}
}
func TestLiveCollectionAlwaysDisclosesHistoryLimit(t *testing.T) {
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		all := strings.Join(args, " ")
		if name != "docker" || !strings.Contains(all, "--context work events --since") || !strings.Contains(all, "--until") {
			t.Fatalf("%s %v", name, args)
		}
		return []byte(event("a", "oom", 100)), nil
	}
	r := Collect(context.Background(), "work", time.Minute, 20, run)
	if r.Outcome != "partial" || r.Events != 1 || len(r.Diagnostics) == 0 {
		t.Fatalf("%+v", r)
	}
}
