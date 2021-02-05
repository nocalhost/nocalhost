package envsubst

import (
	"bufio"
	"io/ioutil"
	"nocalhost/internal/nhctl/envsubst/parse"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strings"
)

func Render(tmpl string, envFilePath string) (string, error) {
	if envFilePath == "" {
		return parse.New("string", [][]string{os.Environ()},
			&parse.Restrictions{NoUnset: false, NoEmpty: false}).Parse(tmpl)
	} else {
		return parse.New("string", [][]string{os.Environ(), readEnvFile(envFilePath)},
			&parse.Restrictions{NoUnset: false, NoEmpty: false}).Parse(tmpl)
	}
}

func readEnvFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.ErrorE(err, "Error while opening file "+filename)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	var envFiles []string

	for scanner.Scan() {
		text := scanner.Text()
		if !strings.ContainsAny(text, "=") || strings.HasPrefix(text, "#") {
			continue
		}
		envFiles = append(envFiles, text)
	}

	return envFiles
}

// Supports specify multiple env source
func StringRestrictedSpecifyEnv(s string, noUnset, noEmpty bool, envs [][]string) (string, error) {
	return parse.New("string", envs,
		&parse.Restrictions{NoUnset: noUnset, NoEmpty: noEmpty}).Parse(s)
}

// StringRestricted returns the parsed template string after processing it.
// If the parser encounters invalid input, or a restriction is violated, it returns
// an error describing the failure.
// Errors on first failure or returns a collection of failures if failOnFirst is false
func StringRestricted(s string, noUnset, noEmpty bool) (string, error) {
	return parse.New("string", [][]string{os.Environ()},
		&parse.Restrictions{NoUnset: noUnset, NoEmpty: noEmpty}).Parse(s)
}

// Bytes returns the bytes represented by the parsed template after processing it.
// If the parser encounters invalid input, it returns an error describing the failure.
func Bytes(b []byte) ([]byte, error) {
	return BytesRestricted(b, false, false)
}

// BytesRestricted returns the bytes represented by the parsed template after processing it.
// If the parser encounters invalid input, or a restriction is violated, it returns
// an error describing the failure.
func BytesRestricted(b []byte, noUnset, noEmpty bool) ([]byte, error) {
	s, err := parse.New("bytes", [][]string{os.Environ()},
		&parse.Restrictions{NoUnset: noUnset, NoEmpty: noEmpty}).Parse(string(b))
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// ReadFile call io.ReadFile with the given file name.
// If the call to io.ReadFile failed it returns the error; otherwise it will
// call envsubst.Bytes with the returned content.
func ReadFile(filename string) ([]byte, error) {
	return ReadFileRestricted(filename, false, false)
}

// ReadFileRestricted calls io.ReadFile with the given file name.
// If the call to io.ReadFile failed it returns the error; otherwise it will
// call envsubst.Bytes with the returned content.
func ReadFileRestricted(filename string, noUnset, noEmpty bool) ([]byte, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return BytesRestricted(b, noUnset, noEmpty)
}
