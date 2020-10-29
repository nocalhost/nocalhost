package go_client
import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func Init() *kubernetes.Clientset {
	var kubeconfig string
	//if home := homedir.HomeDir(); home != "" {
	//	kubeconfig = filepath.Join(home, ".kube", "config")
	//}
	kubeconfig = "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset
}