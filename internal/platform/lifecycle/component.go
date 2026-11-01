package lifecycle

// Component represents a startable/stoppable infrastructure component.
type Component interface {
	Name() string
	Start() error
	Stop() error
}
