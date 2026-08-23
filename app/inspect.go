package app

// Bindings returns the registered type names in registration order.
func (c *Container) Bindings() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	names := make([]string, 0, len(c.order))
	for _, t := range c.order {
		names = append(names, t.String())
	}
	return names
}

// Resolved returns the type names resolved during the container's lifetime, in
// resolution order.
func (c *Container) Resolved() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	names := make([]string, 0, len(c.resolved))
	for _, t := range c.resolved {
		names = append(names, t.String())
	}
	return names
}

// Locked reports whether the container has locked (first resolution happened).
func (c *Container) Locked() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.locked
}
