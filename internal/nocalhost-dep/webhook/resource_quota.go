package webhook
//
//import (
//	"context"
//	"github.com/golang/glog"
//	authorizationv1 "k8s.io/api/authorization/v1"
//	corev1 "k8s.io/api/core/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/client-go/kubernetes"
//	"k8s.io/client-go/rest"
//	"k8s.io/client-go/tools/clientcmd"
//	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
//	"sync"
//	"time"
//)
//
//func getSaByUid(uid string) *corev1.ServiceAccount {
//	sa := listSaCached()[uid]
//	return &sa
//}
//
//// todo supports auto load
//func listSaCached() map[string]corev1.ServiceAccount {
//	now := time.Now()
//	if sasCached != nil && now.Before(sasExpTime) {
//		return sasCached
//	}
//
//	sasLock.Lock()
//	defer sasLock.Unlock()
//
//	if sasCached != nil && now.Before(sasExpTime) {
//		return sasCached
//	}
//
//	glog.Infof("Refreshing service account cache")
//
//	list, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
//	if err != nil {
//		glog.Fatal("Error while listing namespace")
//	}
//
//	result := map[string]corev1.ServiceAccount{}
//
//	if list != nil {
//		for _, ns := range list.Items {
//			var gorLock sync.Mutex
//			go func(nsName string) {
//				for _, sa := range listSaFromNs(nsName) {
//					gorLock.Lock()
//					result[string(sa.UID)] = sa
//					gorLock.Unlock()
//				}
//			}(ns.Name)
//		}
//	}
//
//	// for cluster sa
//	for _, sa := range listSaFromNs("") {
//		result[string(sa.UID)] = sa
//	}
//
//	sasCached = result
//	sasExpTime = now.Add(time.Minute * 5)
//	return sasCached
//}
//
//func listSaFromNs(ns string) []corev1.ServiceAccount {
//	saList, err := clientset.CoreV1().
//		// means all ns
//		ServiceAccounts(ns).
//		List(context.TODO(), metav1.ListOptions{})
//	if err != nil {
//		glog.Infof("Error while listing sa, ns: %s", ns)
//	}
//	if saList == nil || saList.Items == nil {
//		glog.Infof("Ns: %s don't has any sa", ns)
//		return []corev1.ServiceAccount{}
//	}
//	return saList.Items
//}
//
//
