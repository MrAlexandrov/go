package main

import (
	"bufio"
	"family-tree/models"
	"family-tree/parser"
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

type state struct {
	current *models.Person
	from    *models.Person
}

func performStep(current *models.Person, from *models.Person, step rune) []*models.Person {
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

func traversePath(start *models.Person, path string) []*models.Person {
	states := []state{{current: start, from: nil}}

	for _, step := range path {
		stateChan := make(chan state, len(states))
		resultChan := make(chan []state, len(states))

		for _, s := range states {
			stateChan <- s
		}
		close(stateChan)

		numWorkers := max(1, min(len(states), 10))

		g := new(errgroup.Group)
		for range numWorkers {
			g.Go(func() error {
				var localStates []state

				for s := range stateChan {
					nextPersons := performStep(s.current, s.from, step)

					for _, person := range nextPersons {
						localStates = append(localStates, state{
							current: person,
							from:    s.current,
						})
					}
				}

				if len(localStates) > 0 {
					resultChan <- localStates
				}
				return nil
			})
		}

		go func() {
			g.Wait()
			close(resultChan)
		}()

		var nextStates []state
		for localStates := range resultChan {
			nextStates = append(nextStates, localStates...)
		}

		states = nextStates
	}

	seen := make(map[*models.Person]bool)
	var result []*models.Person
	var mutex sync.Mutex

	g := new(errgroup.Group)
	for _, s := range states {
		g.Go(func() error {
			person := s.current

			mutex.Lock()
			defer mutex.Unlock()

			if person != start && !seen[person] {
				seen[person] = true
				result = append(result, person)
			}
			return nil
		})
	}

	g.Wait()

	return result
}

type relationResult struct {
	relation parser.Relation
	found    []*models.Person
}

func processRelationsConcurrently(person *models.Person, relations []parser.Relation) []relationResult {
	results := make([]relationResult, len(relations))

	g := new(errgroup.Group)
	for i, rel := range relations {
		g.Go(func() error {
			found := traversePath(person, rel.Path)
			results[i] = relationResult{
				relation: rel,
				found:    found,
			}
			return nil
		})
	}
	g.Wait()

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

	results := processRelationsConcurrently(person, relations)

	outputChan := make(chan string, tree.GetPeopleNumber())

	var g errgroup.Group
	for _, result := range results {
		g.Go(func() error {
			if len(result.found) == 0 {
				outputChan <- fmt.Sprintf("%s и %s: нет", result.relation.MaleTerm, result.relation.FemaleTerm)
				return nil
			}

			for _, p := range result.found {
				var term string
				if p.Gender == models.Male {
					term = result.relation.MaleTerm
				} else {
					term = result.relation.FemaleTerm
				}
				outputChan <- fmt.Sprintf("%s: %s", term, p.Name)
			}
			return nil
		})
	}

	go func() {
		g.Wait()
		close(outputChan)
	}()

	for line := range outputChan {
		fmt.Println(line)
	}
}
