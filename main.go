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

// State represents a traversal state with current person and previous person
type State struct {
	Current *models.Person
	From    *models.Person
}

// TraversePath traverses the family tree following the given path
func TraversePath(start *models.Person, path string) []*models.Person {
	states := []State{{Current: start, From: nil}}

	for _, step := range path {
		var nextStates []State

		for _, state := range states {
			cur := state.Current
			from := state.From

			add := func(p *models.Person) {
				if p != nil && p != from {
					nextStates = append(nextStates, State{Current: p, From: cur})
				}
			}

			switch step {
			case 'P': // Parents
				p1, p2 := cur.GetParents()
				add(p1)
				add(p2)

			case 'C': // Children
				for _, child := range cur.GetChildren() {
					add(child)
				}

			case 'S': // Spouse
				add(cur.GetSpouse())

			case 'W': // Wife (female spouse)
				sp := cur.GetSpouse()
				if sp != nil && sp.Gender == models.Female {
					add(sp)
				}

			case 'H': // Husband (male spouse)
				sp := cur.GetSpouse()
				if sp != nil && sp.Gender == models.Male {
					add(sp)
				}
			}
		}

		states = nextStates
	}

	// Deduplicate results
	seen := make(map[*models.Person]bool)
	var result []*models.Person

	for _, state := range states {
		person := state.Current
		if person != start && !seen[person] {
			seen[person] = true
			result = append(result, person)
		}
	}

	return result
}

// RelationResult holds the result of processing a single relation
type RelationResult struct {
	Relation parser.Relation
	Found    []*models.Person
}

// ProcessRelationsConcurrently processes all relations concurrently using goroutines
func ProcessRelationsConcurrently(person *models.Person, relations []parser.Relation) []RelationResult {
	results := make([]RelationResult, len(relations))
	var wg sync.WaitGroup

	// Process each relation in a separate goroutine
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

	// Step 1: Parse people file (all persons created concurrently)
	if err := parser.ParsePeopleFile("people.txt", tree); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка парсинга people.txt: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Parse connections file (marriages and children processed in parallel)
	if err := parser.ParseConnectionsFile("connections.txt", tree); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка парсинга connections.txt: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Parse relations file (all relation definitions parsed concurrently)
	relations, err := parser.ParseRelationsFile("relations.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка парсинга relations.txt: %v\n", err)
		os.Exit(1)
	}

	// Read query from user
	fmt.Print("Введите имя: ")
	reader := bufio.NewReader(os.Stdin)
	query, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка чтения ввода: %v\n", err)
		os.Exit(1)
	}

	// Trim whitespace and newlines
	query = strings.TrimSpace(query)

	// Find person
	person := tree.FindPerson(query)
	if person == nil {
		fmt.Printf("Человек \"%s\" не найден.\n", query)
		os.Exit(1)
	}

	fmt.Printf("\nРодственники для %s:\n", query)

	// Process all relations concurrently
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
