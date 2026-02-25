package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"family-tree/models"
)

// Relation represents a family relationship definition
type Relation struct {
	Path       string
	MaleTerm   string
	FemaleTerm string
}

// ParsePeopleFile parses the people file and creates all persons
// Uses goroutines to parse each person concurrently
func ParsePeopleFile(filename string, tree *models.FamilyTree) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	// Read all lines first
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \r\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "(М)") || strings.Contains(line, "(Ж)") {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Parse all persons concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(lines))

	for _, line := range lines {
		wg.Add(1)
		lineCopy := line
		go func(l string) {
			defer wg.Done()
			if err := parsePerson(l, tree); err != nil {
				errChan <- err
			}
		}(lineCopy)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// ParseConnectionsFile parses marriages and parent-child relationships
// Marriages and children can be processed fully in parallel since all persons exist
func ParseConnectionsFile(filename string, tree *models.FamilyTree) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer file.Close()

	var marriages, children []string
	scanner := bufio.NewScanner(file)

	// Read all lines first
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \r\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Contains(line, "<->") {
			marriages = append(marriages, line)
		} else if strings.Contains(line, "->") {
			children = append(children, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Parse marriages first (must be done before children)
	var wg sync.WaitGroup
	errChan := make(chan error, len(marriages))

	for _, line := range marriages {
		wg.Add(1)
		lineCopy := line
		go func(l string) {
			defer wg.Done()
			if err := parseMarriage(l, tree); err != nil {
				errChan <- err
			}
		}(lineCopy)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	// Now parse children relationships (can be done fully in parallel)
	errChan = make(chan error, len(children))

	for _, line := range children {
		wg.Add(1)
		lineCopy := line
		go func(l string) {
			defer wg.Done()
			if err := parseChild(l, tree); err != nil {
				errChan <- err
			}
		}(lineCopy)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// parsePerson parses a person line and adds them to the tree
func parsePerson(line string, tree *models.FamilyTree) error {
	parenIdx := strings.LastIndex(line, " (")
	if parenIdx == -1 {
		return nil
	}

	name := strings.TrimSpace(line[:parenIdx])
	isMale := strings.Contains(line, "(М)")

	gender := models.Female
	if isMale {
		gender = models.Male
	}

	tree.AddPerson(name, gender)
	return nil
}

// parseMarriage parses a marriage relationship
func parseMarriage(line string, tree *models.FamilyTree) error {
	before, after, ok := strings.Cut(line, "<->")
	if !ok {
		return nil
	}

	name1 := strings.TrimSpace(before)
	name2 := strings.TrimSpace(after)

	p1 := tree.FindPerson(name1)
	p2 := tree.FindPerson(name2)

	if p1 != nil && p2 != nil {
		tree.AddCouple(p1, p2)
	}

	return nil
}

// parseChild parses a parent-child relationship
func parseChild(line string, tree *models.FamilyTree) error {
	before, after, ok := strings.Cut(line, "->")
	if !ok {
		return nil
	}

	parentName := strings.TrimSpace(before)
	childName := strings.TrimSpace(after)

	parent := tree.FindPerson(parentName)
	child := tree.FindPerson(childName)

	if parent == nil || child == nil {
		return nil
	}

	couple := parent.GetCouple()
	if couple != nil {
		child.SetParentCouple(couple)
		couple.AddChild(child)
	}

	return nil
}

// ParseRelationsFile parses the relations definition file
// Uses goroutines to parse lines concurrently
func ParseRelationsFile(filename string) ([]Relation, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \r\n")
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Parse relations concurrently
	relations := make([]Relation, len(lines))
	var wg sync.WaitGroup
	var mutex sync.Mutex
	errChan := make(chan error, len(lines))

	for i, line := range lines {
		wg.Add(1)
		go func(idx int, l string) {
			defer wg.Done()

			rel, err := parseRelation(l)
			if err != nil {
				errChan <- err
				return
			}

			mutex.Lock()
			relations[idx] = rel
			mutex.Unlock()
		}(i, line)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	// Filter out empty relations
	var result []Relation
	for _, rel := range relations {
		if rel.Path != "" {
			result = append(result, rel)
		}
	}

	return result, nil
}

// parseRelation parses a single relation line
func parseRelation(line string) (Relation, error) {
	spaceIdx := strings.Index(line, " ")
	if spaceIdx == -1 {
		return Relation{}, nil
	}

	path := line[:spaceIdx]

	openIdx := strings.Index(line[spaceIdx:], "(")
	closeIdx := strings.Index(line[spaceIdx:], ")")

	if openIdx == -1 || closeIdx == -1 {
		return Relation{}, nil
	}

	openIdx += spaceIdx
	closeIdx += spaceIdx

	terms := line[openIdx+1 : closeIdx]
	before, after, ok := strings.Cut(terms, "|")

	if !ok {
		return Relation{}, nil
	}

	maleTerm := before
	femaleTerm := after

	return Relation{
		Path:       path,
		MaleTerm:   maleTerm,
		FemaleTerm: femaleTerm,
	}, nil
}
