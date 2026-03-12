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

func (familyTree *FamilyTree) AddPerson(name string, gender Gender) *Person {
	familyTree.mutex.Lock()
	defer familyTree.mutex.Unlock()

	person := NewPerson(name, gender)
	familyTree.persons = append(familyTree.persons, person)
	familyTree.byName[name] = person
	return person
}

func (familyTree *FamilyTree) AddCouple(p1, p2 *Person) *Couple {
	familyTree.mutex.Lock()
	defer familyTree.mutex.Unlock()

	couple := NewCouple(p1, p2)
	familyTree.couples = append(familyTree.couples, couple)
	p1.SetCouple(couple)
	p2.SetCouple(couple)
	return couple
}

func (familyTree *FamilyTree) FindPerson(name string) *Person {
	familyTree.mutex.RLock()
	defer familyTree.mutex.RUnlock()

	return familyTree.byName[name]
}

func (familyTree *FamilyTree) GetAllPersons() map[string]*Person {
	familyTree.mutex.RLock()
	defer familyTree.mutex.RUnlock()

	result := make(map[string]*Person, len(familyTree.byName))
	maps.Copy(result, familyTree.byName)
	return result
}

func (familyTree *FamilyTree) GetPeopleNumber() int {
	familyTree.mutex.RLock()
	defer familyTree.mutex.RUnlock()

	return len(familyTree.persons)
}
