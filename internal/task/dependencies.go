package task

import "sort"

// FindBlockedTasks returns task IDs that are blocked by the given task.
func FindBlockedTasks(taskID string, tasks map[string]Task) []string {
	var blocks []string
	for id, t := range tasks {
		for _, blocker := range t.BlockedBy {
			if blocker == taskID {
				blocks = append(blocks, id)
				break
			}
		}
	}
	sort.Strings(blocks)
	return blocks
}
