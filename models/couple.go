package models

import "sync"

// Couple represents a married pair and their children
type Couple struct {
	First    *Person
	Second   *Person
	Children []*Person
	mutex    sync.RWMutex // Protects Children slice for concurrent access
}

// NewCouple creates a new Couple instance
func NewCouple(first, second *Person) *Couple {
	return &Couple{
		First:    first,
		Second:   second,
		Children: make([]*Person, 0),
	}
}

// GetSpouseOf returns the spouse of the given person in this couple
func (c *Couple) GetSpouseOf(p *Person) *Person {
	if p == c.First {
		return c.Second
	}
	if p == c.Second {
		return c.First
	}
	return nil
}

// AddChild adds a child to this couple (thread-safe)
func (c *Couple) AddChild(child *Person) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Children = append(c.Children, child)
}

// GetChildren returns a copy of children slice (thread-safe)
func (c *Couple) GetChildren() []*Person {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Return a copy to prevent race conditions
	result := make([]*Person, len(c.Children))
	copy(result, c.Children)
	return result
}
