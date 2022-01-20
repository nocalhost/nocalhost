package envsubst

import (
	"fmt"
	"nocalhost/internal/nhctl/fp"
	"os"
	"testing"
)

func initialEnv() {
	os.Setenv("CODING_GIT_URL", "git@e.coding.net:nocalhost/nocalhost.git")
	os.Setenv(
		"SYNC_FILE_PATTERN", `
     - ./nocalhost
     - ./foo**bar
     - *.jar`,
	)
	os.Setenv("PRIORITY", "2")
}

func TestUnSubst(t *testing.T) {
	result, err := Render(
		LocalFileRenderItem{fp.NewFilePath("testdata/nocalhost.yaml")},
		nil,
	)
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	expected := fp.NewFilePath("testdata/nocalhost_default_result.yaml").ReadFile()
	if result != expected || err != nil {
		t.Errorf("got >>>>\n\t%v\nexpected >>>>\n\t%v", result, expected)
	}
}

func TestSubst(t *testing.T) {
	initialEnv()

	result, err := Render(
		LocalFileRenderItem{fp.NewFilePath("testdata/nocalhost.yaml")},
		nil,
	)
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	expected := fp.NewFilePath("testdata/nocalhost_subst_result.yaml").ReadFile()
	if result != expected || err != nil {
		t.Errorf("got >>>>\n\t%v\nexpected >>>>\n\t%v", result, expected)
	}
}

func TestSubstWithMultiEnv(t *testing.T) {
	initialEnv()

	result, err := Render(
		LocalFileRenderItem{fp.NewFilePath("testdata/nocalhost.yaml")},
		fp.NewFilePath("testdata/.env"),
	)
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	expected := fp.NewFilePath("testdata/nocalhost_subst_multi_env_result.yaml").ReadFile()
	if result != expected || err != nil {
		t.Errorf("got >>>>\n\t%v\nexpected >>>>\n\t%v", result, expected)
	}
}
