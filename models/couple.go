package models

import "sync"

type Couple struct {
	first    *Person // Male
	second   *Person // Female
	children []*Person
	mutex    sync.RWMutex // Защищает детей от cuncurrent доступа
}

func NewCouple(first, second *Person) *Couple {
	return &Couple{
		first:  first,
		second: second,
	}
}

func (c *Couple) GetSpouseOf(p *Person) *Person {
	if p == c.first {
		return c.second
	}
	if p == c.second {
		return c.first
	}
	return nil
}

func (c *Couple) AddChild(child *Person) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.children = append(c.children, child)
}

func (c *Couple) GetChildren() []*Person {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := make([]*Person, len(c.children))
	copy(result, c.children)
	return result
}
