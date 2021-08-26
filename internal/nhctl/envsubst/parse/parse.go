/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

// Most of the code in this package taken from golang/text/template/parse
package parse

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/fp"
	"strconv"
	"strings"
)

// A mode value is a set of flags (or 0). They control parser behavior.
type Mode int

// Mode for parser behaviour
const (
	Quick            Mode = iota // stop parsing after first error encoutered and return
	AllErrors                    // report all errors
	IncludeSensitive = "\\|"
	IncludeSeparator = "|"
	Include          = "_INCLUDE_"
	Indent           = "nindent"
	AbsSign          = "#/---//---//Nocalhost//---//---/"

	ErrTmpl = `
#  ======    WARN    ======
#  Error occur while resolve following includation:
#  Err       :  %s
#  File      :  %s
#  Statement :  %s
#  Route     :  %s
#  ======    WARN    ======
`
)

// The restrictions option controls the parsring restriction.
type Restrictions struct {
	NoUnset bool
	NoEmpty bool
}

// Restrictions specifier
var (
	Relaxed = &Restrictions{false, false}
	NoEmpty = &Restrictions{false, true}
	NoUnset = &Restrictions{true, false}
	Strict  = &Restrictions{true, true}
)

// Parser type initializer
type Parser struct {
	Name     string // name of the processing template
	Env      []Env
	Restrict *Restrictions
	Mode     Mode
	// parsing state;
	lex       *lexer
	token     [3]item // three-token lookahead
	peekCount int
	nodes     []Node
}

type Includation struct {
	include  string
	filePath *fp.FilePathEnhance
	indent   int
	err      error
}

// New allocates a new Parser with the given name.
// envs means support multi source env[]
// the priority rely by the order
func New(name string, envs [][]string, r *Restrictions) *Parser {
	var env []Env

	for _, e := range envs {
		env = append(env, e)
	}

	return &Parser{
		Name:     name,
		Env:      env,
		Restrict: r,
	}
}

func (p *Parser) ParseWithoutIncludation(text string) (string, error) {
	return p.Parse(text, "", []string{})
}

// Parse parses the given string.
func (p *Parser) Parse(text string, absPath string, hasBeenInclude []string) (string, error) {
	hasBeenInclude = append(hasBeenInclude, absPath)

	p.lex = lex(text)
	// Build internal array of all unset or empty vars here
	var errs []error
	// clean parse state
	p.nodes = make([]Node, 0)
	p.peekCount = 0
	if err := p.parse(); err != nil {
		switch p.Mode {
		case Quick:
			return "", err
		case AllErrors:
			errs = append(errs, err)
		}
	}
	var out string

	out += fmt.Sprintln(fmt.Sprintf("%s%s", AbsSign, absPath))
	for _, node := range p.nodes {
		k, v, err := node.String()

		if err != nil {
			switch p.Mode {
			case Quick:
				return "", errors.Wrap(err, "")
			case AllErrors:
				errs = append(errs, err)
			}
		}

		// Resolve the includation
		// (1) resolve dependency
		// (2) parse the include syntax
		// (3) read the file content from include
		// (4) insert indent if needed
		if k == Include {
			includation := parseIncludation(absPath, v)

			currentAbsPath := includation.filePath.Abs()
			circularDependency, route := circularDependency(currentAbsPath, hasBeenInclude)
			if circularDependency {
				out += fmt.Sprintf(ErrTmpl, errors.New("circular dependency found"), currentAbsPath, v, route)
				continue
			}

			if includation.err != nil {
				out += fmt.Sprintf(ErrTmpl, includation.err.Error(), currentAbsPath, v, route)
				continue
			}

			content, err := includation.filePath.ReadFileCompel()
			if err != nil {
				out += fmt.Sprintf(ErrTmpl, err.Error(), currentAbsPath, v, route)
				continue
			}

			include, err := p.Parse(content, currentAbsPath, hasBeenInclude)
			if err != nil {
				out += fmt.Sprintf(ErrTmpl, err.Error(), currentAbsPath, v, route)
				continue
			}
			v = insertIndent(includation.indent, include)
			v += fmt.Sprintln(fmt.Sprintf("\n%s%s", AbsSign, absPath))
		}

		out += fmt.Sprint(v)
	}


	if len(errs) > 0 {
		var b strings.Builder
		for i, err := range errs {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(err.Error())
		}
		return "", errors.New(b.String())
	}
	return out, nil
}

// insert the indent for each line
func insertIndent(indent int, text string) string {
	var result string
	EOL := "\n"

	// need to prevent different indent cause by multi line
	if strings.Contains(text, EOL) {
		text = EOL + text
	}

	if indent > 0 {

		// to add indent by replace \n
		indentReplacement := ""
		for i := 0; i < indent; i++ {
			indentReplacement += " "
		}

		result = strings.ReplaceAll(text, EOL, EOL+indentReplacement)
	} else {
		result = text
	}

	return result
}

// Detect whether has circular dependency
// true shows it is and the route is the dependency tracing
func circularDependency(currentAbsPath string, hasBeenInclude []string) (bool, string) {
	circularDependency := false
	route := "\n"

	for _, absPath := range hasBeenInclude {
		if absPath == currentAbsPath {
			circularDependency = true

			route += fmt.Sprintf("# ┌-->  %s\n# |        ↓ [Include]\n", absPath)
		} else {
			if circularDependency {
				route += fmt.Sprintf("# |     %s\n# |        ↓ [Include]\n", absPath)
			} else {
				route += fmt.Sprintf("#       %s\n#          ↓ [Include]\n", absPath)
			}
		}
	}

	if circularDependency {
		route += fmt.Sprintf("# └--- %s", currentAbsPath)
	} else {
		route += fmt.Sprintf("#      %s", currentAbsPath)
	}

	return circularDependency, route
}

// todo: need to refactor
// parse the include syntax
func parseIncludation(basePath, include string) *Includation {
	// prevent split the user's path
	start := strings.LastIndex(include, IncludeSensitive)

	// prefix is use to escape '|' due to it is a sensitive character
	var pathPrefix string
	if start == -1 {
		// means do not need prefix
		start = 0
	} else {
		// 2 = len(\|)
		start += 2
	}

	// pathPrefix make sure all '\|' is managed, to avoid interference operation resolve
	pathPrefix = strings.ReplaceAll(include[:start], IncludeSensitive, IncludeSeparator)
	pathAndOper := strings.Split(include[start:], IncludeSeparator)

	switch len(pathAndOper) {
	case 1:
		include = strings.TrimSpace(include)
		return &Includation{
			include:  include,
			filePath: fp.NewFilePath(basePath).RelOrAbs("../").RelOrAbs(include),
			indent:   0,
		}
	case 2:
		path := strings.TrimSpace(pathPrefix + pathAndOper[0])
		oper := pathAndOper[1]

		originOperations := strings.Split(oper, " ")

		// remove unnecessary statement
		var operations []string
		for _, operation := range originOperations {
			if operation != "" {
				operations = append(operations, operation)
			}
		}

		operLen := len(operations)

		switch operLen {

		// without indent
		case 0:
			return &Includation{
				include:  include,
				filePath: fp.NewFilePath(basePath).RelOrAbs("../").RelOrAbs(path),
				indent:   0,
			}

		// try to resolve Indent
		case 2:
			if Indent == operations[0] {
				indent, err := strconv.Atoi(operations[1])
				if err != nil || indent < 0 {
					return &Includation{
						include: include,
						err:     errors.New("Can not parse the indent, please make sure it's a positive integer: " + oper),
					}
				}

				return &Includation{
					include:  include,
					filePath: fp.NewFilePath(basePath).RelOrAbs("../").RelOrAbs(path),
					indent:   indent,
				}

			} else {
				return &Includation{
					include: include,
					err:     errors.New("Do not support such syntax yet: " + oper),
				}
			}

		// for now, do not support more syntax
		default:
			return &Includation{
				include: include,
				err:     errors.New("Can not resolve the include syntax: " + oper),
			}
		}

	default:
		return &Includation{
			include: include,
			err: errors.New(
				"Can not resolve the include syntax " +
					"(may contains multi '|', if your path sensitive character contains " +
					"'|', use `\\|` to replace it): " + include,
			),
		}
	}
}

// parse is the top-level parser for the template.
// It runs to EOF and return an error if something isn't right.
func (p *Parser) parse() error {
Loop:
	for {
		switch t := p.next(); t.typ {
		case itemEOF:
			break Loop
		case itemError:
			return p.errorf(t.val)
		case itemVariable:
			varNode := NewVariable(strings.TrimPrefix(t.val, "$"), p.Env, p.Restrict)
			p.nodes = append(p.nodes, varNode)
		case itemLeftDelim:
			if p.peek().typ == itemVariable {
				n, err := p.action()
				if err != nil {
					return err
				}
				p.nodes = append(p.nodes, n)
				continue
			}
			fallthrough
		default:
			textNode := NewText(t.val)
			p.nodes = append(p.nodes, textNode)
		}
	}
	return nil
}

// Parse substitution. first item is a variable.
func (p *Parser) action() (Node, error) {
	var expType itemType
	var defaultNode Node
	varNode := NewVariable(p.next().val, p.Env, p.Restrict)
Loop:
	for {
		switch t := p.next(); t.typ {
		case itemRightDelim:
			break Loop
		case itemError:
			return nil, p.errorf(t.val)
		case itemVariable:
			defaultNode = NewVariable(strings.TrimPrefix(t.val, "$"), p.Env, p.Restrict)
		case itemText:
			n := NewText(t.val)
		Text:
			for {
				switch p.peek().typ {
				case itemRightDelim, itemError, itemEOF:
					break Text
				default:
					// patch to accept all kind of chars
					n.Text += p.next().val
				}
			}
			defaultNode = n
		default:
			expType = t.typ
		}
	}
	return &SubstitutionNode{NodeSubstitution, expType, varNode, defaultNode}, nil
}

func (p *Parser) errorf(s string) error {
	return errors.New(s)
}

// next returns the next token.
func (p *Parser) next() item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lex.nextItem()
	}
	return p.token[p.peekCount]
}

// backup backs the input stream up one token.
func (p *Parser) backup() {
	p.peekCount++
}

// peek returns but does not consume the next token.
func (p *Parser) peek() item {
	if p.peekCount > 0 {
		return p.token[p.peekCount-1]
	}
	p.peekCount = 1
	p.token[0] = p.lex.nextItem()
	return p.token[0]
}
