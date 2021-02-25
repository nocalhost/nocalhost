package envsubst

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func initialEnv() {
	os.Setenv("CODING_GIT_URL", "git@e.coding.net:nocalhost/nocalhost.git")
	os.Setenv("SYNC_FILE_PATTERN", `
     - ./nocalhost
     - ./foo**bar
     - *.jar`)
	os.Setenv("PRIORITY", "2")
}

func readFile(filename string) string {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return ""
	}
	return string(b)
}

func TestUnSubst(t *testing.T) {
	content := readFile("testdata/nocalhost.tmpl")

	result, err := Render(content, "")
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	fexpected := readFile("testdata/nocalhost_default.result")
	if result != fexpected || err != nil {
		t.Errorf("got >>>>\n\t%v\nexpected >>>>\n\t%v", result, fexpected)
	}
}

func TestSubst(t *testing.T) {
	initialEnv()
	content := readFile("testdata/nocalhost.tmpl")

	result, err := Render(content, "")
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	fexpected := string(readFile("testdata/nocalhost_subst.result"))
	if result != fexpected || err != nil {
		t.Errorf("got >>>>\n\t%v\nexpected >>>>\n\t%v", result, fexpected)
	}
}

func TestSubstWithMultiEnv(t *testing.T) {
	initialEnv()
	content := readFile("testdata/nocalhost.tmpl")

	result, err := Render(content, "testdata/.env")
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	fexpected := string(readFile("testdata/nocalhost_subst_multi_env.result"))
	if result != fexpected || err != nil {
		t.Errorf("got >>>>\n\t%v\nexpected >>>>\n\t%v", result, fexpected)
	}
}
