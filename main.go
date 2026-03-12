package main

import (
	"bufio"
	"family-tree/models"
	"family-tree/parser"
	"fmt"
	"os"
	"strings"
	"sync"
)

type State struct {
	Current *models.Person
	From    *models.Person
}

func PerformStep(current *models.Person, from *models.Person, step rune) []*models.Person {
	var result []*models.Person

	add := func(p *models.Person) {
		if p != nil && p != from {
			result = append(result, p)
		}
	}

	switch step {
	case 'P': // Parents
		p1, p2 := current.GetParents()
		add(p1)
		add(p2)

	case 'C': // Children
		for _, child := range current.GetChildren() {
			add(child)
		}

	case 'S': // Spouse
		add(current.GetSpouse())

	case 'W': // Wife (female spouse)
		sp := current.GetSpouse()
		if sp != nil && sp.Gender == models.Female {
			add(sp)
		}

	case 'H': // Husband (male spouse)
		sp := current.GetSpouse()
		if sp != nil && sp.Gender == models.Male {
			add(sp)
		}
	}

	return result
}

func TraversePath(start *models.Person, path string) []*models.Person {
	states := []State{{Current: start, From: nil}}

	for _, step := range path {
		stateChan := make(chan State, len(states))
		resultChan := make(chan []State, len(states))

		for _, state := range states {
			stateChan <- state
		}
		close(stateChan)

		var wg sync.WaitGroup
		numWorkers := max(1, min(len(states), 10))

		for range numWorkers {
			wg.Go(func() {
				var localStates []State

				for state := range stateChan {
					nextPersons := PerformStep(state.Current, state.From, step)

					for _, person := range nextPersons {
						localStates = append(localStates, State{
							Current: person,
							From:    state.Current,
						})
					}
				}

				if len(localStates) > 0 {
					resultChan <- localStates
				}
			})
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		var nextStates []State
		for localStates := range resultChan {
			nextStates = append(nextStates, localStates...)
		}

		states = nextStates
	}

	seen := make(map[*models.Person]bool)
	var result []*models.Person
	var mutex sync.Mutex

	var wg sync.WaitGroup
	for _, state := range states {
		wg.Add(1)
		go func(s State) {
			defer wg.Done()
			person := s.Current

			mutex.Lock()
			defer mutex.Unlock()

			if person != start && !seen[person] {
				seen[person] = true
				result = append(result, person)
			}
		}(state)
	}
	wg.Wait()

	return result
}

type RelationResult struct {
	Relation parser.Relation
	Found    []*models.Person
}

func ProcessRelationsConcurrently(person *models.Person, relations []parser.Relation) []RelationResult {
	results := make([]RelationResult, len(relations))
	var wg sync.WaitGroup

	for i, rel := range relations {
		wg.Add(1)
		go func(idx int, relation parser.Relation) {
			defer wg.Done()

			found := TraversePath(person, relation.Path)
			results[idx] = RelationResult{
				Relation: relation,
				Found:    found,
			}
		}(i, rel)
	}

	wg.Wait()
	return results
}

func main() {
	tree := models.NewFamilyTree()

	if err := parser.ParsePeopleFile("people.txt", tree); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка парсинга people.txt: %v\n", err)
		os.Exit(1)
	}

	if err := parser.ParseConnectionsFile("connections.txt", tree); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка парсинга connections.txt: %v\n", err)
		os.Exit(1)
	}

	relations, err := parser.ParseRelationsFile("relations.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка парсинга relations.txt: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Введите имя: ")
	reader := bufio.NewReader(os.Stdin)
	query, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка чтения ввода: %v\n", err)
		os.Exit(1)
	}

	query = strings.TrimSpace(query)

	person := tree.FindPerson(query)
	if person == nil {
		fmt.Printf("Человек \"%s\" не найден.\n", query)
		os.Exit(1)
	}

	fmt.Printf("\nРодственники для %s:\n", query)

	results := ProcessRelationsConcurrently(person, relations)

	outputChan := make(chan string, tree.GetPeopleNumber())
	var wg sync.WaitGroup

	for _, result := range results {
		wg.Add(1)
		go func(res RelationResult) {
			defer wg.Done()

			if len(res.Found) == 0 {
				outputChan <- fmt.Sprintf("%s и %s: нет", res.Relation.MaleTerm, res.Relation.FemaleTerm)
				return
			}

			for _, p := range res.Found {
				var term string
				if p.Gender == models.Male {
					term = res.Relation.MaleTerm
				} else {
					term = res.Relation.FemaleTerm
				}
				outputChan <- fmt.Sprintf("%s: %s", term, p.Name)
			}
		}(result)
	}

	go func() {
		wg.Wait()
		close(outputChan)
	}()

	for line := range outputChan {
		fmt.Println(line)
	}
}
