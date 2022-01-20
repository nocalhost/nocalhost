package clientgoutils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"sync"
	"time"
)

var (
	// holds all the server-supported resources that cannot be discovered by clients.
	// i.e. users and groups for the impersonate verb
	nonStandardResourceNames = sets.NewString("users", "groups")

	AllVerbs = []string{"get", "list", "watch", "create", "patch", "delete"}

	authCheckManagerMapping     = make(map[string]*authCheckManager)
	authCheckManagerMappingLock sync.Mutex

	PermissionDenied = errors.New("Permission Denied")
)

func CheckForResource(kubeConfigContent, namespace string, verbs []string, passWhenTimeout bool, resources ...string) error {
	return getManagerCached(kubeConfigContent).
		AuthCheckForResource(namespace, verbs, passWhenTimeout, resources...)
}

func getManagerCached(kubeConfigContent string) *authCheckManager {
	authCheckManagerMappingLock.Lock()
	defer authCheckManagerMappingLock.Unlock()

	var mgr *authCheckManager

	if manager, ok := authCheckManagerMapping[kubeConfigContent]; ok {
		mgr = manager
	} else {
		path := k8sutils.GetOrGenKubeConfigPath(kubeConfigContent)
		mgr = NewAuthCheckManager(path)
		authCheckManagerMapping[kubeConfigContent] = mgr
	}

	return mgr
}

func (a *authCheckManager) AuthCheckForResource(namespace string, verbs []string, passWhenTimeout bool, resources ...string) error {
	if !a.initSuccess {
		log.Errorf("AuthCheckManager init fail before, so skip auth check...")
		return nil
	}

	if verbs == nil || len(verbs) == 0 {
		verbs = AllVerbs
	}

	var acs = make([]*AuthChecker, 0)
	for _, resource := range resources {
		acs = append(acs, &AuthChecker{ResourceArg: resource, Verb: verbs})
	}

	return a.doAuthCheck(namespace, passWhenTimeout, acs...)
}

func (a *authCheckManager) doFastCheck(namespace string) bool {
	a.fastCheckLock.Lock()
	defer a.fastCheckLock.Unlock()
	if b, ok := a.fastCheckMap[namespace]; ok {
		return b
	}

	ra := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Group:     "*",
		Verb:      "*",
		Name:      "*",
		Version:   "*",
		Resource:  "*",
	}

	result, err := a.auth(ra)
	if err != nil {
		return false
	}

	a.fastCheckMap[namespace] = result.Status.Allowed
	log.Logf("Fast Auth for namespace: %v", a.fastCheckMap[namespace])
	return a.fastCheckMap[namespace]
}

func (a *authCheckManager) doAuthCheck(namespace string, passWhenTimeout bool, authCheckers ...*AuthChecker) error {
	a.client.namespace = namespace

	forbiddenCheckers := make([]*authorizationv1.ResourceAttributes, 0)
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	errChan := make(chan error)
	okChan := make(chan int)
	fastCheckChan := make(chan int)

	// fast check, check for * * * * *
	// if pass, no need to check the specified entry
	go func() {
		if a.doFastCheck(namespace) {
			fastCheckChan <- 0
		}
	}()

	if authCheckers != nil {
		for _, checker := range authCheckers {

			for _, verb := range checker.Verb {
				verb := verb

				wg.Add(1)
				go func() {
					defer wg.Done()
					r := a.client.ResourceFor(checker.ResourceArg, true)

					// first try load from cache inner authorizationMap

					ra := &authorizationv1.ResourceAttributes{
						Namespace: namespace,
						Group:     r.Group,
						Verb:      verb,
						Name:      checker.Name,
						Version:   checker.Version,
						Resource:  r.Resource,
					}

					cacheKey := fmt.Sprintf(
						"%s-%s-%s-%s-%s-%s",
						namespace, r.Group, verb, checker.Name, checker.Version, r.Resource,
					)

					authAllow := false
					if allow, ok := a.authorizationMap.Load(cacheKey); ok && !allow.(bool) {
						authAllow = allow.(bool)
					} else {
						resp, err := a.auth(ra)

						if err != nil {
							errChan <- err
							return
						}

						authAllow = resp.Status.Allowed
						a.authorizationMap.Store(
							cacheKey,
							authAllow,
						)
					}

					if !authAllow {
						lock.Lock()
						defer lock.Unlock()
						forbiddenCheckers = append(forbiddenCheckers, ra)
					}
				}()
			}
		}
	}

	go func() {
		wg.Wait()
		okChan <- 0
	}()

	select {
	case <-fastCheckChan:
	case <-okChan:
		// if check over 5 second, stopping to check the result
	case <-time.NewTicker(time.Second * 5).C:
		if !passWhenTimeout {
			return errors.New("Time out when auth check, please try again!")
		}
	case e := <-errChan:
		return e
	}

	if len(forbiddenCheckers) > 0 {
		marshal, _ := json.Marshal(forbiddenCheckers)
		return errors.Wrap(
			PermissionDenied,
			fmt.Sprintf(
				"Permission denied when auth check! "+
					"please make sure you have such permission in current namespace: \n%s", marshal,
			),
		)
	}

	return nil
}

func (a *authCheckManager) auth(ra *authorizationv1.ResourceAttributes) (*authorizationv1.SelfSubjectAccessReview, error) {
	arg := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: ra,
		},
	}

	return a.client.ClientSet.AuthorizationV1().SelfSubjectAccessReviews().
		Create(context.TODO(), arg, metav1.CreateOptions{})
}

func NewAuthCheckManager(kubeConfigPath string) *authCheckManager {
	success := true

	client, err := NewClientGoUtils(kubeConfigPath, "")
	if err != nil {
		success = false
		log.ErrorE(err, "Error while init auth checker ")
	}

	return &authCheckManager{
		initSuccess: success,
		client:      client,

		gvrCache:         map[string]schema.GroupVersionResource{},
		lock:             sync.Mutex{},
		authorizationMap: sync.Map{},

		fastCheckMap:  map[string]bool{},
		fastCheckLock: sync.Mutex{},
	}
}

type authCheckManager struct {
	initSuccess bool

	client *ClientGoUtils

	gvrCache map[string]schema.GroupVersionResource
	lock     sync.Mutex

	fastCheckLock    sync.Mutex
	fastCheckMap     map[string]bool
	authorizationMap sync.Map

	initLatch sync.Once
}

type AuthChecker struct {
	Verb        []string
	Name        string
	Version     string
	ResourceArg string
}
