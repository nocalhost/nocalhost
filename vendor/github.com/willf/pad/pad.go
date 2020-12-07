/*
Package pad provides left-padding functionality


*/
package pad

import "strings"

func times(str string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(str, n)
}

// Left left-pads the string with pad up to len runes
// len may be exceeded if
func Left(str string, length int, pad string) string {
	return times(pad, length-len(str)) + str
}

// Right right-pads the string with pad up to len runes
func Right(str string, length int, pad string) string {
	return str + times(pad, length-len(str))
}
