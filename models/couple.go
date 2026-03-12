package models

import "sync"

type Couple struct {
	First    *Person // Male
	Second   *Person // Female
	Children []*Person
	mutex    sync.RWMutex // Protects Children slice for concurrent access
}

func NewCouple(first, second *Person) *Couple {
	return &Couple{
		First:    first,
		Second:   second,
		Children: make([]*Person, 0),
	}
}

func (c *Couple) GetSpouseOf(p *Person) *Person {
	if p == c.First {
		return c.Second
	}
	if p == c.Second {
		return c.First
	}
	return nil
}

func (c *Couple) AddChild(child *Person) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Children = append(c.Children, child)
}

func (c *Couple) GetChildren() []*Person {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := make([]*Person, len(c.Children))
	copy(result, c.Children)
	return result
}
