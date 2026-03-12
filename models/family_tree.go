package models

import (
	"maps"
	"sync"
)

type FamilyTree struct {
	persons []*Person
	couples []*Couple
	byName  map[string]*Person
	mutex   sync.RWMutex
}

func NewFamilyTree() *FamilyTree {
	return &FamilyTree{
		persons: make([]*Person, 0),
		couples: make([]*Couple, 0),
		byName:  make(map[string]*Person),
	}
}

func (ft *FamilyTree) AddPerson(name string, gender Gender) *Person {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	person := NewPerson(name, gender)
	ft.persons = append(ft.persons, person)
	ft.byName[name] = person
	return person
}

func (ft *FamilyTree) AddCouple(p1, p2 *Person) *Couple {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	couple := NewCouple(p1, p2)
	ft.couples = append(ft.couples, couple)
	p1.SetCouple(couple)
	p2.SetCouple(couple)
	return couple
}

func (ft *FamilyTree) FindPerson(name string) *Person {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()

	return ft.byName[name]
}

func (ft *FamilyTree) GetAllPersons() map[string]*Person {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()

	result := make(map[string]*Person, len(ft.byName))
	maps.Copy(result, ft.byName)
	return result
}

func (ft *FamilyTree) GetPeopleNumber() int {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()

	return len(ft.persons)
}
