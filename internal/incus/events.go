package incus

// docs:
// https://linuxcontainers.org/incus/docs/main/events/#supported-life-cycle-events

const eventTypeLifecycle = "lifecycle"

const (
	EventInstanceStarted   = "instance-started"
	EventInstanceStopped   = "instance-stopped"
	EventInstanceShutdown  = "instance-shutdown"
	EventInstanceDeleted   = "instance-deleted"
	EventInstanceRenamed   = "instance-renamed"
	EventInstanceRestarted = "instance-restarted"
)

type InstanceEvent struct {
	Name    string
	Project string
	Action  string
}
