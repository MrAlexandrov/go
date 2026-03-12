package models

import (
	"maps"
	"sync"
)

// FamilyTree represents the entire family tree structure
type FamilyTree struct {
	persons []*Person
	couples []*Couple
	byName  map[string]*Person
	mutex   sync.RWMutex
}

// NewFamilyTree creates a new FamilyTree instance
func NewFamilyTree() *FamilyTree {
	return &FamilyTree{
		persons: make([]*Person, 0),
		couples: make([]*Couple, 0),
		byName:  make(map[string]*Person),
	}
}

// AddPerson adds a new person to the family tree
func (familyTree *FamilyTree) AddPerson(name string, gender Gender) *Person {
	familyTree.mutex.Lock()
	defer familyTree.mutex.Unlock()

	person := NewPerson(name, gender)
	familyTree.persons = append(familyTree.persons, person)
	familyTree.byName[name] = person
	return person
}

// AddCouple creates a couple relationship between two people
func (familyTree *FamilyTree) AddCouple(p1, p2 *Person) *Couple {
	familyTree.mutex.Lock()
	defer familyTree.mutex.Unlock()

	couple := NewCouple(p1, p2)
	familyTree.couples = append(familyTree.couples, couple)
	p1.SetCouple(couple)
	p2.SetCouple(couple)
	return couple
}

// FindPerson finds a person by name
func (familyTree *FamilyTree) FindPerson(name string) *Person {
	familyTree.mutex.RLock()
	defer familyTree.mutex.RUnlock()

	return familyTree.byName[name]
}

// GetAllPersons returns all persons in the tree
func (familyTree *FamilyTree) GetAllPersons() map[string]*Person {
	familyTree.mutex.RLock()
	defer familyTree.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*Person, len(familyTree.byName))
	maps.Copy(result, familyTree.byName)
	return result
}

func (familyTree *FamilyTree) GetPeopleNumber() int {
	familyTree.mutex.RLock()
	defer familyTree.mutex.RUnlock()

	return len(familyTree.persons)
}
