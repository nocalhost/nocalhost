package webhook

import (
	"context"
	"github.com/golang/glog"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
	"sync"
	"time"
)

var sasCached map[string]corev1.ServiceAccount = nil
var sasLock sync.Mutex
var sasExpTime = time.Now()

func getSaByUid(uid string) *corev1.ServiceAccount {
	sa := listSaCached()[uid]
	return &sa
}

// todo supports auto load
func listSaCached() map[string]corev1.ServiceAccount {
	now := time.Now()
	if sasCached != nil && now.Before(sasExpTime) {
		return sasCached
	}

	sasLock.Lock()
	defer sasLock.Unlock()

	if sasCached != nil && now.Before(sasExpTime) {
		return sasCached
	}

	glog.Infof("Refreshing service account cache")

	list, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		glog.Fatal("Error while listing namespace")
	}


	result := map[string]corev1.ServiceAccount{}

	if list != nil {
		for _, ns := range list.Items {
			var gorLock sync.Mutex
			go func(nsName string) {
				for _, sa := range listSaFromNs(nsName) {
					gorLock.Lock()
					result[string(sa.UID)] = sa
					gorLock.Unlock()
				}
			}(ns.Name)
		}
	}

	// for cluster sa
	for _, sa := range listSaFromNs("") {
		result[string(sa.UID)] = sa
	}

	sasCached = result
	sasExpTime = now.Add(time.Minute * 5)
	return sasCached
}

func listSaFromNs(ns string) []corev1.ServiceAccount {
	saList, err := clientset.CoreV1().
		// means all ns
		ServiceAccounts(ns).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		glog.Infof("Error while listing sa, ns: %s", ns)
	}
	if saList == nil || saList.Items == nil {
		glog.Infof("Ns: %s don't has any sa", ns)
		return []corev1.ServiceAccount{}
	}
	return saList.Items
}

func isClusterAdmin(sa *corev1.ServiceAccount) (bool, error) {
	if len(sa.Secrets) == 0 {
		return false, nil
	}

	secret, err := clientset.CoreV1().Secrets(sa.Namespace).Get(context.TODO(), sa.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return false, err
	}

	KubeConfigYaml, err, _ := setupcluster.NewDevKubeConfigReader(secret, config.Host, sa.Namespace).GetCA().GetToken().AssembleDevKubeConfig().ToYamlString()
	if err != nil {
		return false, err
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(KubeConfigYaml))
	if err != nil {
		return false, nil
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return false, nil
	}

	arg := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: "*",
				Group:     "*",
				Verb:      "*",
				Name:      "*",
				Version:   "*",
				Resource:  "*",
			},
		},
	}

	response, err := cs.AuthorizationV1().SelfSubjectAccessReviews().Create(context.TODO(), arg, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return response.Status.Allowed, nil
}
