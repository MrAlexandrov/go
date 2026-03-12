package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"family-tree/models"

	"golang.org/x/sync/errgroup"
)

// Relation represents a family relationship definition
type Relation struct {
	Path       string
	MaleTerm   string
	FemaleTerm string
}

// ParsePeopleFile parses the people file and creates all persons
// Uses pipeline pattern: read → filter → parse → add to tree
func ParsePeopleFile(filename string, tree *models.FamilyTree) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer file.Close()

	// Stage 1: Read lines from file
	lineChan := make(chan string, 100)
	go func() {
		defer close(lineChan)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), " \r\n")
			if line != "" && !strings.HasPrefix(line, "#") {
				lineChan <- line
			}
		}
	}()

	// Stage 2: Filter valid person lines
	validLineChan := make(chan string, 100)
	go func() {
		defer close(validLineChan)
		for line := range lineChan {
			if strings.Contains(line, "(М)") || strings.Contains(line, "(Ж)") {
				validLineChan <- line
			}
		}
	}()

	// Stage 3: Parse persons concurrently (fan-out)
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Launch multiple workers
	numWorkers := 5
	for range numWorkers {
		wg.Go(func() {
			for line := range validLineChan {
				if err := ParsePerson(line, tree); err != nil {
					errChan <- err
					return
				}
			}
		})
	}

	// Wait for all workers and close error channel
	go func() {
		wg.Wait()
		close(errChan)
	}()

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
	g := new(errgroup.Group)
	for _, line := range marriages {
		g.Go(func() error {
			return ParseMarriage(line, tree)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	g = new(errgroup.Group)
	for _, line := range children {
		g.Go(func() error {
			return ParseChild(line, tree)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

// ParsePerson parses a person line and adds them to the tree
func ParsePerson(line string, tree *models.FamilyTree) error {
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

// ParseMarriage parses a marriage relationship
func ParseMarriage(line string, tree *models.FamilyTree) error {
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

// ParseChild parses a parent-child relationship
func ParseChild(line string, tree *models.FamilyTree) error {
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

			rel, err := ParseRelation(l)
			if err != nil {
				errChan <- err
				return
			}

			mutex.Lock()
			relations[idx] = rel
			mutex.Unlock()
		}(i, line)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

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

// ParseRelation parses a single relation line
func ParseRelation(line string) (Relation, error) {
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
