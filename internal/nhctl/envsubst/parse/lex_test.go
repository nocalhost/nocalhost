package parse

import (
	"testing"
)

type lexTest struct {
	name  string
	input string
	items []item
}

var (
	tEOF       = item{itemEOF, 0, ""}
	tPlus      = item{itemPlus, 0, ""}
	tDash      = item{itemDash, 0, "-"}
	tEquals    = item{itemEquals, 0, "="}
	tColEquals = item{itemColonEquals, 0, ":="}
	tColDash   = item{itemColonDash, 0, ":-"}
	tColPlus   = item{itemColonPlus, 0, ":+"}
	tLeft      = item{itemLeftDelim, 0, "${"}
	tRight     = item{itemRightDelim, 0, "}"}
)

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"text", "hello", []item{
		{itemText, 0, "hello"},
		tEOF,
	}},
	{"var", "$hello", []item{
		{itemVariable, 0, "$hello"},
		tEOF,
	}},
	{"2 vars", "$hello $world", []item{
		{itemVariable, 0, "$hello"},
		{itemText, 0, " "},
		{itemVariable, 0, "$world"},
		tEOF,
	}},
	{"substitution-1", "bar ${BAR}", []item{
		{itemText, 0, "bar "},
		tLeft,
		{itemVariable, 0, "BAR"},
		tRight,
		tEOF,
	}},
	{"substitution-2", "bar ${BAR:=baz}", []item{
		{itemText, 0, "bar "},
		tLeft,
		{itemVariable, 0, "BAR"},
		tColEquals,
		{itemText, 0, "b"},
		{itemText, 0, "a"},
		{itemText, 0, "z"},
		tRight,
		tEOF,
	}},
	{"substitution-3", "bar ${BAR:=$BAZ}", []item{
		{itemText, 0, "bar "},
		tLeft,
		{itemVariable, 0, "BAR"},
		tColEquals,
		{itemVariable, 0, "$BAZ"},
		tRight,
		tEOF,
	}},
	{"substitution-4", "bar ${BAR:=$BAZ} foo", []item{
		{itemText, 0, "bar "},
		tLeft,
		{itemVariable, 0, "BAR"},
		tColEquals,
		{itemVariable, 0, "$BAZ"},
		tRight,
		{itemText, 0, " foo"},
		tEOF,
	}},
	{"closing brace error", "hello-${world", []item{
		{itemText, 0, "hello-"},
		tLeft,
		{itemVariable, 0, "world"},
		{itemError, 0, "closing brace expected"},
	}},
	{"escaping $$var", "hello $$HOME", []item{
		{itemText, 0, "hello "},
		{itemText, 7, "$"},
		{itemText, 8, "HOME"},
		tEOF,
	}},
	{"escaping $${subst}", "hello $${HOME}", []item{
		{itemText, 0, "hello "},
		{itemText, 7, "$"},
		{itemText, 8, "{HOME}"},
		tEOF,
	}},
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test)
		if !equal(items, test.items, false) {
			t.Errorf("%s:\ninput\n\t%q\ngot\n\t%+v\nexpected\n\t%v", test.name, test.input, items, test.items)
		}
	}
}

// collect gathers the emitted items into a slice.
func collect(t *lexTest) (items []item) {
	l := lex(t.input)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}
	return
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
	}
	return true
}
