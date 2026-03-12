package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"family-tree/models"

	"golang.org/x/sync/errgroup"
)

type Relation struct {
	Path       string
	MaleTerm   string
	FemaleTerm string
}

func ParsePeopleFile(filename string, tree *models.FamilyTree) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer file.Close()

	// Pipeline pattern
	var scanErr error
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
		scanErr = scanner.Err()
	}()

	validLineChan := make(chan string, 100)
	go func() {
		defer close(validLineChan)
		for line := range lineChan {
			if strings.Contains(line, "(М)") || strings.Contains(line, "(Ж)") {
				validLineChan <- line
			}
		}
	}()

	// Worker pool pattern
	numWorkers := 5
	g := new(errgroup.Group)
	for range numWorkers {
		g.Go(func() error {
			for line := range validLineChan {
				if err := ParsePerson(line, tree); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return scanErr
}

func ParseConnectionsFile(filename string, tree *models.FamilyTree) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer file.Close()

	var marriages, children []string
	scanner := bufio.NewScanner(file)

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

	relations := make([]Relation, len(lines))

	g := new(errgroup.Group)
	for i, line := range lines {
		g.Go(func() error {
			rel, err := ParseRelation(line)
			if err != nil {
				return err
			}

			relations[i] = rel
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var result []Relation
	for _, rel := range relations {
		if rel.Path != "" {
			result = append(result, rel)
		}
	}

	return result, nil
}

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
