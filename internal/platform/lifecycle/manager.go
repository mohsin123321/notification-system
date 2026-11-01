package lifecycle

import "log"

// Manager handles ordered startup and reverse-order shutdown of components.
type Manager struct {
	components []Component
}

func NewManager() *Manager {
	return &Manager{components: make([]Component, 0)}
}

func (m *Manager) Register(components ...Component) {
	m.components = append(m.components, components...)
}

func (m *Manager) StartAll() error {
	for _, c := range m.components {
		log.Printf("starting %s", c.Name())
		if err := c.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) StopAll() {
	for i := len(m.components) - 1; i >= 0; i-- {
		c := m.components[i]
		log.Printf("stopping %s", c.Name())
		if err := c.Stop(); err != nil {
			log.Printf("error stopping %s: %v", c.Name(), err)
		}
	}
}
