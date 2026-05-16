package incus

// docs:
// https://linuxcontainers.org/incus/docs/main/events/#supported-life-cycle-events

const eventTypeLifecycle = "lifecycle"

var EventsPurgeCache []string = []string{
	"instance-stopped",
	"instance-shutdown",
	"instance-deleted",
	"instance-renamed", // `old_name`: purge
}

var EventsUpdateCache []string = []string{
	"instance-started",
	"instance-renamed", // new name -> cache
	"instance-restarted",
	// "instance-updated", -> add only if tracked/cached field can change
}

type InstanceEvent struct {
	Name    string
	Project string
	Action  string
}
