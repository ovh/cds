package scheduler

import "testing"

func TestSchedulerRun(t *testing.T) {
	exs, err := SchedulerRun()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Has prepare %v", exs)
}
