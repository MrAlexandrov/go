package models

type Gender int

const (
	Male Gender = iota
	Female
)

type Person struct {
	Name         string
	Gender       Gender
	couple       *Couple
	parentCouple *Couple
}

func NewPerson(name string, gender Gender) *Person {
	return &Person{
		Name:   name,
		Gender: gender,
	}
}

func (p *Person) GetSpouse() *Person {
	if p.couple == nil {
		return nil
	}
	return p.couple.GetSpouseOf(p)
}

func (p *Person) GetParents() (*Person, *Person) {
	if p.parentCouple == nil {
		return nil, nil
	}
	return p.parentCouple.First, p.parentCouple.Second
}

func (p *Person) GetChildren() []*Person {
	if p.couple == nil {
		return nil
	}
	return p.couple.GetChildren()
}

func (p *Person) SetCouple(c *Couple) {
	p.couple = c
}

func (p *Person) SetParentCouple(c *Couple) {
	p.parentCouple = c
}

func (p *Person) GetCouple() *Couple {
	return p.couple
}

func (p *Person) GetParentCouple() *Couple {
	return p.parentCouple
}
