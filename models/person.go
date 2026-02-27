package models

// Gender represents the gender of a person
type Gender int

const (
	Male Gender = iota
	Female
)

// Person represents an individual in the family tree
type Person struct {
	Name         string
	Gender       Gender
	couple       *Couple
	parentCouple *Couple
}

// NewPerson creates a new Person instance
func NewPerson(name string, gender Gender) *Person {
	return &Person{
		Name:   name,
		Gender: gender,
	}
}

// GetSpouse returns the spouse of this person, or nil if not married
func (p *Person) GetSpouse() *Person {
	if p.couple == nil {
		return nil
	}
	return p.couple.GetSpouseOf(p)
}

// GetParents returns both parents of this person
func (p *Person) GetParents() (*Person, *Person) {
	if p.parentCouple == nil {
		return nil, nil
	}
	return p.parentCouple.First, p.parentCouple.Second
}

// GetChildren returns all children of this person (thread-safe)
func (p *Person) GetChildren() []*Person {
	if p.couple == nil {
		return nil
	}
	return p.couple.GetChildren()
}

// SetCouple sets the couple this person belongs to
func (p *Person) SetCouple(c *Couple) {
	p.couple = c
}

// SetParentCouple sets the parent couple of this person
func (p *Person) SetParentCouple(c *Couple) {
	p.parentCouple = c
}

// GetCouple returns the couple this person belongs to
func (p *Person) GetCouple() *Couple {
	return p.couple
}

// GetParentCouple returns the parent couple of this person
func (p *Person) GetParentCouple() *Couple {
	return p.parentCouple
}
