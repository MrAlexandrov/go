package models

// Couple represents a married pair and their children
type Couple struct {
	First    *Person
	Second   *Person
	Children []*Person
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

// AddChild adds a child to this couple
func (c *Couple) AddChild(child *Person) {
	c.Children = append(c.Children, child)
}
