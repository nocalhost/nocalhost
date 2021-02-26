package envsubst

import (
	"fmt"
	"nocalhost/internal/nhctl/fp"
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

func TestUnSubst(t *testing.T) {
	result, err := Render(fp.NewFilePath("testdata/nocalhost.tmpl"), nil)
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	fexpected := fp.NewFilePath("testdata/nocalhost_default.result").ReadFile()
	if result != fexpected || err != nil {
		t.Errorf("got >>>>\n\t%v\nexpected >>>>\n\t%v", result, fexpected)
	}
}

func TestSubst(t *testing.T) {
	initialEnv()

	result, err := Render(fp.NewFilePath("testdata/nocalhost.tmpl"), nil)
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	fexpected := fp.NewFilePath("testdata/nocalhost_subst.result").ReadFile()
	if result != fexpected || err != nil {
		t.Errorf("got >>>>\n\t%v\nexpected >>>>\n\t%v", result, fexpected)
	}
}

func TestSubstWithMultiEnv(t *testing.T) {
	initialEnv()

	result, err := Render(fp.NewFilePath("testdata/nocalhost.tmpl"), fp.NewFilePath("testdata/.env"))
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	fexpected :=  fp.NewFilePath("testdata/nocalhost_subst_multi_env.result").ReadFile()
	if result != fexpected || err != nil {
		t.Errorf("got >>>>\n\t%v\nexpected >>>>\n\t%v", result, fexpected)
	}
}
