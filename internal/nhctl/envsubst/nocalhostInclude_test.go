package envsubst

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"nocalhost/internal/nhctl/fp"
	"os"
	"reflect"
	"testing"
)

func initialIncludeEnv() {
	os.Setenv("DETAILS_DEV_IMAGE", "RUBBBBBBY!!")
}

func TestInclude(t *testing.T) {
	initialIncludeEnv()

	result, err := Render(
		LocalFileRenderItem{fp.NewFilePath("testdata/include/nocalhostIncludeRenderTmpl.yaml")},
		nil,
	)
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	expected := fp.NewFilePath("testdata/include/rendered.yaml").ReadFile()

	err = IsSameYaml(result, expected)
	if err != nil {
		t.Error(err)
	}
}

func TestCircularInclude(t *testing.T) {
	initialIncludeEnv()

	result, err := Render(
		LocalFileRenderItem{fp.NewFilePath("testdata/circularDependency/renderTmpl.yaml")},
		nil,
	)
	if err != nil {
		fmt.Printf("%+v", err)
		t.Error(err)
	}

	expected := fp.NewFilePath("testdata/circularDependency/rendered.yaml").ReadFile()

	err = IsSameYaml(result, expected)
	if err != nil {
		t.Error(err)
	}
}

func IsSameYaml(src, dst string) error {
	var resultMap map[string]interface{}
	err := yaml.Unmarshal([]byte(src), &resultMap)

	if err != nil {
		return errors.New(fmt.Sprintf("Parse error >>> \n %s \n Err: %s", src, err))
	}

	var expectedMap map[string]interface{}
	err = yaml.Unmarshal([]byte(dst), &expectedMap)

	if err != nil {
		return errors.New(fmt.Sprintf("Parse expected error >>> \n %s \n Err: %s", dst, err))
	}

	if !reflect.DeepEqual(resultMap, expectedMap) {
		return errors.New(fmt.Sprintf("got >>>>>>>>>>>>>>>>>>>>\n%v\nexpected >>>>>>>>>>>>>>>>>>>>\n%v", src, dst))
	}

	return nil
}
