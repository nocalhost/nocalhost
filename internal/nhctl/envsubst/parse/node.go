/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package parse

import (
	"fmt"
	"github.com/pkg/errors"
)

type Node interface {
	Type() NodeType
	String() (string, string, error)
}

// NodeType identifies the type of a node.
type NodeType int

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeText NodeType = iota
	NodeSubstitution
	NodeVariable
)

type TextNode struct {
	NodeType
	Text string
}

func NewText(text string) *TextNode {
	return &TextNode{NodeText, text}
}

func (t *TextNode) String() (string, string, error) {
	return "", t.Text, nil
}

type VariableNode struct {
	NodeType
	Ident    string
	Env      []Env
	Restrict *Restrictions
}

func NewVariable(ident string, env []Env, restrict *Restrictions) *VariableNode {
	return &VariableNode{NodeVariable, ident, env, restrict}
}

func (t *VariableNode) String() (string, string, error) {
	if err := t.validateNoUnset(); err != nil {
		return t.Ident, "", err
	}

	var value *string = nil

	for _, env := range t.Env {
		value = env.Get(t.Ident)

		// distinguish nil and "" is necessary because we allow user to specify "" in env file
		if value != nil {
			break
		}
	}

	var result string
	if value == nil {
		result = ""
	} else {
		result = *value
	}

	if err := t.validateNoEmpty(result); err != nil { // ???
		return t.Ident, "", errors.Wrap(err, "")
	}
	return t.Ident, result, nil
}

func (t *VariableNode) isSet() bool {
	for _, env := range t.Env {
		if env.Has(t.Ident) {
			return true
		}
	}
	return false
}

func (t *VariableNode) validateNoUnset() error {
	if t.Restrict.NoUnset && !t.isSet() {
		return fmt.Errorf("variable ${%s} not set", t.Ident)
	}
	return nil
}

func (t *VariableNode) validateNoEmpty(value string) error {
	if t.Restrict.NoEmpty && value == "" && t.isSet() {
		return fmt.Errorf("variable ${%s} set but empty", t.Ident)
	}
	return nil
}

type SubstitutionNode struct {
	NodeType
	ExpType  itemType
	Variable *VariableNode
	Default  Node // Default could be variable or text
}

func (t *SubstitutionNode) String() (string, string, error) {
	if t.ExpType >= itemPlus && t.Default != nil {
		switch t.ExpType {
		case itemColonDash, itemColonEquals:
			if k, v, _ := t.Variable.String(); v != "" {
				return k, v, nil
			}
			return t.StringWithDefault()
		case itemPlus, itemColonPlus:
			if t.Variable.isSet() {
				return t.StringWithDefault()
			}
			return t.Variable.Ident, "", nil
		default:
			if !t.Variable.isSet() {
				return t.StringWithDefault()
			}
		}
	}
	return t.Variable.String()
}

func (t *SubstitutionNode) StringWithDefault() (k string, v string, err error) {
	k, _, _ = t.Variable.String()
	_, v, err = t.Default.String()
	return
}
