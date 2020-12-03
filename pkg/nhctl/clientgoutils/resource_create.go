package clientgoutils

import (
	"errors"
	"fmt"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func (c *ClientGoUtils) ApplyForCreate(files []string, namespace string, continueOnError bool) error {
	if len(files) == 0 {
		return errors.New("files must not be nil")
	}
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	builder := f.NewBuilder()
	//ns, _ := c.GetDefaultNamespace()
	validate, err := f.Validator(true)
	if err != nil {
		return err
	}
	filenames := resource.FilenameOptions{
		Filenames: files,
		Kustomize: "",
		Recursive: false,
	}
	if continueOnError {
		builder.ContinueOnError()
	}
	result := builder.Unstructured().
		Schema(validate).
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(true, &filenames).
		//LabelSelectorParam(o.Selector).
		Flatten().Do()

	if result == nil {
		return errors.New("result is nil")
	}
	if result.Err() != nil {
		return result.Err()
	}

	infos, err := result.Infos()
	if err != nil {
		return err
	}

	if len(infos) == 0 {
		return errors.New("no result info")
	}

	for _, info := range infos {
		helper := resource.NewHelper(info.Client, info.Mapping)
		obj, err := helper.Create(info.Namespace, true, info.Object)
		if err != nil {
			if continueOnError {
				continue
			}
			return err
		}
		info.Refresh(obj, true)
		fmt.Printf("%s/%s created\n", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name)
	}

	return nil
}
