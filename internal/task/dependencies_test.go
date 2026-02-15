package task

import "testing"

func TestFindBlockedTasksReturnsSortedMatches(t *testing.T) {
	tasks := map[string]Task{
		"gamma": {BlockedBy: []string{"auth"}},
		"alpha": {BlockedBy: []string{"auth", "other"}},
		"beta":  {BlockedBy: []string{"other"}},
	}

	got := FindBlockedTasks("auth", tasks)
	want := []string{"alpha", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("expected %d results, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected sorted results %v, got %v", want, got)
		}
	}
}

func TestFindBlockedTasksReturnsEachTaskOnce(t *testing.T) {
	tasks := map[string]Task{
		"alpha": {BlockedBy: []string{"auth", "auth"}},
	}

	got := FindBlockedTasks("auth", tasks)
	if len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("expected [alpha], got %v", got)
	}
}
