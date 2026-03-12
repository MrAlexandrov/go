package models

import "sync"

type Couple struct {
	First    *Person // Male
	Second   *Person // Female
	children []*Person
	mutex    sync.RWMutex // Защищает детей от cuncurrent доступа
}

func NewCouple(first, second *Person) *Couple {
	return &Couple{
		First:  first,
		Second: second,
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
	c.children = append(c.children, child)
}

func (c *Couple) GetChildren() []*Person {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := make([]*Person, len(c.children))
	copy(result, c.children)
	return result
}
