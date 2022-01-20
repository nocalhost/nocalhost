/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package resouce_cache

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"sort"
	"strings"
	"sync"
	"time"
)

// cache Searcher for each kubeconfig
var searchMap, _ = simplelru.NewLRU(20, func(_ interface{}, value interface{}) {
	if value != nil {
		if s, ok := value.(*Searcher); ok && s != nil {
			s.Stop()
		}
	}
})
var searchMapLock = &sync.Mutex{}

var clusterMap = make(map[string]bool)
var clusterMapLock = &sync.Mutex{}

// key: generateKey(kubeconfigBytes, namespace) value: []*restmapper.APIGroupResources
var apiGroupResourcesMap sync.Map

type Searcher struct {
	kubeconfigBytes []byte
	//informerFactory        informers.SharedInformerFactory
	dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
	// [string]*meta.RESTMapping
	supportSchemaWithAlias *sync.Map // ResourceType: GvkGvrWithAlias
	SupportSchemaList      []GvkGvrWithAlias
	stopChan               chan struct{}
	// last used this searcher, for release informer resource
	lastUsedTime time.Time
}

func (s *Searcher) GetSupportSchema() *sync.Map {
	return s.supportSchemaWithAlias
}

type GvkGvrWithAlias struct {
	Gvr   schema.GroupVersionResource
	Gvk   schema.GroupVersionKind
	alias []string
	// namespaced indicates if a resource is namespaced or not.
	Namespaced bool
}

func (g *GvkGvrWithAlias) GetFullName() string {
	name := g.Gvr.Resource
	if g.Gvr.Version != "" && g.Gvr.Group != "" {
		name += "." + g.Gvr.Version
		name += "." + g.Gvr.Group
	}
	return name
}

// getSupportedSchema return restMapping of each resource, [string]*meta.RESTMapping
// Key: resourceType
func getSupportedSchema(apiResources []*restmapper.APIGroupResources) ([]GvkGvrWithAlias, sets.String, error) {
	var resourceNeeded = map[string]string{"namespaces": "Namespace.v1."} // deployment/statefulset...
	for _, v := range GroupToTypeMap {
		for _, s := range v.V {
			resourceNeeded[s] = s
		}
	}

	nameToMapping := make([]GvkGvrWithAlias, 0) // []GvkGvrWithAlias

	for _, s := range resourceNeeded {
		gvk, gk := schema.ParseKindArg(s)
		if gvk == nil {
			gvk = &schema.GroupVersionKind{Kind: gk.Group, Group: gk.Group}
		}
		apiR, err := ConvertGvkToApiResource(gvk, apiResources)
		if err == nil {
			ggwa := GvkGvrWithAlias{
				Gvr: schema.GroupVersionResource{
					Group:    gvk.Group,
					Version:  gvk.Version,
					Resource: apiR.Name,
				},
				Gvk: *gvk,
				alias: []string{
					apiR.Name, apiR.SingularName, strings.ToLower(apiR.Kind),
				},
				Namespaced: apiR.Namespaced,
			}
			ggwa.alias = append(ggwa.alias, apiR.ShortNames...)
			ggwa.alias = append(ggwa.alias, ggwa.GetFullName())
			nameToMapping = append(nameToMapping, ggwa)
		}
	}

	if len(nameToMapping) == 0 {
		return nil, nil, errors.New("RestMapping is empty, this should not happened")
	}

	// workloads need to parse app from annotation
	set := sets.NewString()
	for _, s := range GroupToTypeMap[0].V {
		arg, _ := schema.ParseKindArg(s)
		if arg != nil {
			set.Insert(arg.GroupKind().String())
		}
	}
	return nameToMapping, set, nil
}

func ConvertRuntimeObjectToCRD(obj runtime.Object) (*apiextensions.CustomResourceDefinition, error) {
	um, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("Fail to convert to unstructured")
	}
	jsonBytes, err := um.MarshalJSON()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	crdObj := &apiextensions.CustomResourceDefinition{}
	if err = json.Unmarshal(jsonBytes, crdObj); err != nil {
		return nil, errors.WithStack(err)
	}
	return crdObj, nil
}

// todo: support multi versions
func getCrdSchema(client *clientgoutils.ClientGoUtils, apiGroupResources []*restmapper.APIGroupResources) ([]GvkGvrWithAlias, error) {

	crds, err := client.ListResourceInfo("crd")
	if err != nil {
		return nil, err
	}
	nameToMapping := make([]GvkGvrWithAlias, 0)

	for _, crd := range crds {
		crdObj, err := ConvertRuntimeObjectToCRD(crd.Object)
		if err != nil {
			continue
		}

		gs := ConvertCRDToGgwa(crdObj, apiGroupResources)
		if len(gs) > 0 {
			nameToMapping = append(nameToMapping, gs...)
		}
	}

	crdGvk := schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	}
	apiR, err := ConvertGvkToApiResource(&crdGvk, apiGroupResources)
	if err != nil {
		log.Warnf("Failed to convert gvk %v to apiResource", crdGvk)
	} else {
		ggwa := GvkGvrWithAlias{
			Gvr: schema.GroupVersionResource{
				Group:    crdGvk.Group,
				Version:  crdGvk.Version,
				Resource: apiR.Name,
			},
			Gvk: crdGvk,
			alias: []string{
				apiR.Name, apiR.SingularName, strings.ToLower(apiR.Kind),
			},
			Namespaced: false,
		}
		ggwa.alias = append(ggwa.alias, apiR.ShortNames...)
		ggwa.alias = append(ggwa.alias, "crds")
		nameToMapping = append(nameToMapping, ggwa)
	}

	if len(nameToMapping) == 0 {
		return nil, errors.New("RestMapping is empty, this should not happened")
	}
	return nameToMapping, nil
}

func ConvertCRDToGgwa(crdObj *apiextensions.CustomResourceDefinition, agrs []*restmapper.APIGroupResources) []GvkGvrWithAlias {
	result := make([]GvkGvrWithAlias, 0)
	for _, version := range crdObj.Spec.Versions {
		ggwa := GvkGvrWithAlias{
			Gvk: schema.GroupVersionKind{
				Group:   crdObj.Spec.Group,
				Version: version.Name,
				Kind:    crdObj.Spec.Names.Kind,
			},
			alias:      []string{},
			Namespaced: crdObj.Spec.Scope == apiextensions.NamespaceScoped,
		}
		apiR, err := ConvertGvkToApiResource(&ggwa.Gvk, agrs)
		if err != nil {
			log.Warnf("Failed to convert gvk %v to apiResource", ggwa.Gvk)
			continue
		}
		ggwa.Gvr = schema.GroupVersionResource{
			Group:    ggwa.Gvk.Group,
			Version:  ggwa.Gvk.Version,
			Resource: apiR.Name,
		}
		ggwa.alias = append(ggwa.alias, fmt.Sprintf("%s.%s.%s", apiR.Name, ggwa.Gvr.Version, ggwa.Gvr.Group))
		result = append(result, ggwa)
	}
	return result
}

func ConvertGvkToApiResource(gvk *schema.GroupVersionKind, grs []*restmapper.APIGroupResources) (*metav1.APIResource, error) {
	for _, grList := range grs {
		if grList.Group.Name != gvk.Group {
			continue
		}
		for version, resources := range grList.VersionedResources {
			if version != gvk.Version {
				continue
			}
			for _, apiResource := range resources {
				if apiResource.Kind != gvk.Kind {
					continue
				}
				return &apiResource, nil
			}
		}
	}
	return nil, errors.New("Can not convert gvk to gvr")
}

func GetApiGroupResources(kubeBytes []byte, ns string) ([]*restmapper.APIGroupResources, error) {
	clusterKey := generateKey(kubeBytes, ns)
	var gr []*restmapper.APIGroupResources
	v, ok := apiGroupResourcesMap.Load(clusterKey)
	if !ok {
		kubeconfigPath := k8sutils.GetOrGenKubeConfigPath(string(kubeBytes))
		clientUtils, err := clientgoutils.NewClientGoUtils(kubeconfigPath, ns)
		if err != nil {
			return nil, err
		}
		if gr, err = clientUtils.GetAPIGroupResources(); err != nil {
			return nil, err
		}
		apiGroupResourcesMap.Store(clusterKey, gr)
	} else {
		if gr, ok = v.([]*restmapper.APIGroupResources); !ok {
			return nil, errors.New("apiGroupResourcesMap value is not []*restmapper.APIGroupResources")
		}
	}
	return gr, nil
}

// GetSearcherWithLRU GetSearchWithLRU will cache kubeconfig with LRU
func GetSearcherWithLRU(kubeconfigBytes []byte, namespace string) (search *Searcher, err error) {
	defer func() {
		if search != nil {
			search.lastUsedTime = time.Now()
		}
	}()
	clusterKey := generateKey(kubeconfigBytes, namespace)
	searchMapLock.Lock()
	searcher, exist := searchMap.Get(clusterKey)
	searchMapLock.Unlock()
	if !exist || searcher == nil {
		kubeconfigPath := k8sutils.GetOrGenKubeConfigPath(string(kubeconfigBytes))
		clientUtils, err := clientgoutils.NewClientGoUtils(kubeconfigPath, namespace)
		if err != nil {
			return nil, err
		}

		var gr []*restmapper.APIGroupResources
		v, ok := apiGroupResourcesMap.Load(clusterKey)
		if !ok {
			if gr, err = clientUtils.GetAPIGroupResources(); err != nil {
				return nil, err
			}
			apiGroupResourcesMap.Store(clusterKey, gr)
		} else {
			if gr, ok = v.([]*restmapper.APIGroupResources); !ok {
				return nil, errors.New("apiGroupResourcesMap value is not []*restmapper.APIGroupResources")
			}
		}

		newSearcher, err := initSearcher(kubeconfigBytes, namespace, clientUtils, gr)
		if err != nil {
			return nil, err
		}
		searchMapLock.Lock()
		defer searchMapLock.Unlock()
		log.Infof("Search map is len is %d", searchMap.Len()+1)
		clusterKey = generateKey(kubeconfigBytes, namespace)
		if searcher, exist = searchMap.Get(clusterKey); exist && searcher != nil {
			newSearcher.Stop()
			search = searcher.(*Searcher)
			return search, nil
		} else {
			searchMap.Add(clusterKey, newSearcher)
			search = newSearcher
			return search, nil
		}
	}
	search = searcher.(*Searcher)
	err = nil
	return search, err
}

// calculate kubeconfig content's sha value as unique cluster id
func generateKey(kubeconfigBytes []byte, namespace string) string {
	h := sha1.New()
	h.Write(kubeconfigBytes)
	// if it's a cluster admin kubeconfig, then generate key without namespace
	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()
	if _, found := clusterMap[string(kubeconfigBytes)]; found {
		return string(h.Sum(nil))
	} else {
		return string(h.Sum([]byte(namespace)))
	}
}

// initSearcher return a searcher which use informer to cache resource, without cache
func initSearcher(kubeconfigBytes []byte, namespace string, clientUtils *clientgoutils.ClientGoUtils,
	gr []*restmapper.APIGroupResources) (*Searcher, error) {
	log.Infof("InitSearcher for ns: %s", namespace)

	//// default value is flowcontrol.NewTokenBucketRateLimiter(5, 10)
	//config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(10000, 10000)

	//var informerFactory informers.SharedInformerFactory
	var dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
	var err error

	if clientUtils.IsClusterAdmin() {
		dynamicInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(clientUtils.GetDynamicClient(), time.Second*5)
		clusterMapLock.Lock()
		clusterMap[string(kubeconfigBytes)] = true
		clusterMapLock.Unlock()
	} else {
		dynamicInformerFactory = dynamicinformer.NewFilteredDynamicSharedInformerFactory(clientUtils.GetDynamicClient(), time.Second*5, namespace, nil)
	}

	var crdRestMappingList []GvkGvrWithAlias

	if clientUtils.IsClusterAdmin() {
		crdRestMappingList, err = getCrdSchema(clientUtils, gr)
		if err != nil {
			log.WarnE(err, "Failed to get crd schema")
		}
	}

	restMappingList, workloads, err := getSupportedSchema(gr)
	if err != nil {
		return nil, err
	}

	supportedSchema := sync.Map{}
	for _, resource := range restMappingList {
		informer := dynamicInformerFactory.ForResource(resource.Gvr)
		if workloads.Has(resource.Gvk.GroupKind().String()) {
			informer.Informer().AddEventHandler(NewResourceEventHandlerFuncs(informer, kubeconfigBytes, resource.Gvr))
		}

		for _, alias := range resource.alias {
			supportedSchema.Store(alias, resource)
		}
	}

	for _, resource := range crdRestMappingList {
		dynamicInformerFactory.ForResource(resource.Gvr)
		for _, alias := range resource.alias {
			supportedSchema.Store(alias, resource)
		}
	}

	for _, aliases := range crdRestMappingList {
		restMappingList = append(restMappingList, aliases)
	}

	stopCRDChannel := make(chan struct{}, 1)
	dynamicInformerFactory.Start(stopCRDChannel)
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	dynamicInformerFactory.WaitForCacheSync(ctx.Done())

	newSearcher := &Searcher{
		kubeconfigBytes:        kubeconfigBytes,
		dynamicInformerFactory: dynamicInformerFactory,
		supportSchemaWithAlias: &supportedSchema,
		SupportSchemaList:      restMappingList,
		stopChan:               stopCRDChannel,
	}
	return newSearcher, nil
}

// Start wait searcher to close
func (s *Searcher) Start() {
	<-s.stopChan
}

// Stop to stop the searcher
func (s *Searcher) Stop() {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	close(s.stopChan)
}

func (s *Searcher) GetKubeconfigBytes() []byte {
	return s.kubeconfigBytes
}

func (s *Searcher) GetResourceInfo(resourceType string) (GvkGvrWithAlias, error) {
	if value, found := s.supportSchemaWithAlias.Load(strings.ToLower(resourceType)); found && value != nil {
		if restMapping, convert := value.(GvkGvrWithAlias); convert {
			return restMapping, nil
		}
	}
	return GvkGvrWithAlias{}, errors.New(fmt.Sprintf("Can't get restMapping, resource type: %s", resourceType))
}

// e's annotation appName must in appNameRange, otherwise app name is not available
// Get app name from annotation
func getAppName(e interface{}) string {
	object := e.(metav1.Object)
	annotations := object.GetAnnotations()
	if object.GetDeletionTimestamp() != nil || annotations == nil {
		return _const.DefaultNocalhostApplication
	}
	if len(annotations[_const.NocalhostApplicationName]) != 0 {
		return annotations[_const.NocalhostApplicationName]
	}
	if len(annotations[_const.HelmReleaseName]) != 0 {
		return annotations[_const.HelmReleaseName]
	}
	return _const.DefaultNocalhostApplication
}

// vendor/k8s.io/client-go/tools/cache/store.go:99, the reason why using ns/resource to get resource
func nsResource(ns, resourceName string) string {
	return fmt.Sprintf("%s/%s", ns, resourceName)
}

//func SortByNameAsc(item []interface{}) {
//	sort.SliceStable(
//		item, func(i, j int) bool {
//			return item[i].(*unstructured.Unstructured).DeepCopy().GetName() < item[j].(metav1.Object).GetName()
//		},
//	)
//}

func (s *Searcher) Criteria() *criteria {
	return newCriteria(s)
}

type criteria struct {
	search       *Searcher
	resourceType string
	//namespaceScope bool
	resourceName string
	appName      string
	ns           string
	label        map[string]string
	showHidden   bool
}

func newCriteria(search *Searcher) *criteria {
	return &criteria{search: search}
}
func (c *criteria) Namespace(namespace string) *criteria {
	c.ns = namespace
	return c
}

func (c *criteria) AppName(appName string) *criteria {
	c.appName = appName
	return c
}

func (c *criteria) ResourceType(resourceType string) *criteria {
	c.resourceType = resourceType
	return c
}

func (c *criteria) ResourceName(resourceName string) *criteria {
	c.resourceName = resourceName
	return c
}

func (c *criteria) Label(label map[string]string) *criteria {
	c.label = label
	return c
}

func (c *criteria) ShowHidden(showHidden bool) *criteria {
	c.showHidden = showHidden
	return c
}

func (c *criteria) QueryOne() (interface{}, error) {
	query, err := c.Query()
	if err != nil {
		return nil, err
	}
	if len(query) == 0 {
		return nil, errors.New("not found")
	}
	return query[0], nil
}

func (c *criteria) Consume(consumer func([]interface{}) error) error {
	query, err := c.Query()
	if err != nil {
		return err
	}
	return consumer(query)
}

// Query Get data
func (c *criteria) Query() (data []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recover in query")
			e, _ = err.(error)
		}
		if mapping, errs := c.search.GetResourceInfo(c.resourceType); errs == nil {
			for i, d := range data {
				dd := d.(runtime.Object).GetObjectKind()
				if dd.GroupVersionKind().Empty() {
					if ddd, ok := d.(*unstructured.Unstructured); ok {
						deepCopy := ddd.DeepCopy()
						deepCopy.GetObjectKind().SetGroupVersionKind(mapping.Gvk)
						data[i] = deepCopy
						dd.SetGroupVersionKind(mapping.Gvk)
					} else {
						dd.SetGroupVersionKind(mapping.Gvk)
					}
				}
			}
		}
	}()

	if c.search == nil {
		return nil, errors.New("search should not be null")
	}
	if len(c.resourceType) == 0 {
		return nil, errors.New("resource type should not be null")
	}
	mapping, err := c.search.GetResourceInfo(c.resourceType)
	if err != nil {
		return nil, err
	}
	informer := c.search.dynamicInformerFactory.ForResource(mapping.Gvr)
	if informer == nil {
		return nil, errors.New("create informer failed, please check your code")
	}

	if !mapping.Namespaced {
		list := informer.Informer().GetStore().List()
		if len(c.resourceName) != 0 {
			for _, i := range list {
				if i.(metav1.Object).GetName() == c.resourceName {
					return []interface{}{i}, nil
				}
			}
			return []interface{}{}, nil
		}
		iters := make([]interface{}, 0)
		for _, object := range list {
			iters = append(iters, object)
		}
		//SortByNameAsc(iters)
		result := newFilter(iters).
			namespace(c.ns).
			appName(c.appName).
			label(c.label)
		if !c.showHidden {
			result.notLabel(map[string]string{_const.DevWorkloadIgnored: "true"})
		}
		return result.sort().toSlice(), nil
		//return iters, nil
	}

	// if namespace and resourceName is not empty both, using indexer to query data
	if len(c.ns) != 0 && len(c.resourceName) != 0 {
		//item, exists, err := informer.Informer().GetIndexer().GetByKey(nsResource(c.ns, c.resourceName))
		item, exists, err := informer.Informer().GetStore().GetByKey(nsResource(c.ns, c.resourceName))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if !exists {
			return nil, errors.Errorf(
				"not found for resource : %s-%s in namespace: %s", c.resourceType, c.resourceName, c.ns,
			)
		}

		// this is a filter, if appName is empty, just return value
		if len(c.appName) == 0 || c.appName == getAppName(item) {
			data = append(data, item)
			return data, nil
		}
		return
	}

	objs := informer.Informer().GetStore().List()
	iters := make([]interface{}, 0)
	for _, obj := range objs {
		iters = append(iters, obj)
	}
	result := newFilter(iters).
		namespace(c.ns).
		appName(c.appName).
		label(c.label)
	if !c.showHidden {
		result.notLabel(map[string]string{_const.DevWorkloadIgnored: "true"})
	}
	return result.sort().toSlice(), nil
}

type filter struct {
	element []interface{}
}

func newFilter(element []interface{}) *filter {
	return &filter{element: element}
}

func (n *filter) namespace(namespace string) *filter {
	if len(namespace) == 0 {
		return n
	}
	var result []interface{}
	for _, e := range n.element {
		if e.(metav1.Object).GetNamespace() == namespace {
			result = append(result, e)
		}
	}
	n.element = result[0:]
	return n
}

func (n *filter) appName(appName string) *filter {
	if len(appName) == 0 {
		return n
	}
	var result []interface{}
	for _, e := range n.element {
		if getAppName(e) == appName {
			result = append(result, e)
		}
	}
	n.element = result[0:]
	return n
}

// support equals, like: a == b
func (n *filter) label(label map[string]string) *filter {
	n.element = labelSelector(n.element, label, func(v1, v2 string) bool { return v1 == v2 })
	return n
}

// support not equal, like a != b
func (n *filter) notLabel(label map[string]string) *filter {
	n.element = labelSelector(n.element, label, func(v1, v2 string) bool { return v1 != v2 })
	return n
}

func labelSelector(element []interface{}, label map[string]string, f func(string, string) bool) []interface{} {
	var result []interface{}
	for _, e := range element {
		labels := e.(metav1.Object).GetLabels()
		match := true
		for k, v := range label {
			if !f(labels[k], v) {
				match = false
				break
			}
		}
		if match {
			result = append(result, e)
		}
	}
	return result[0:]
}

func (n *filter) sort() *filter {
	sort.SliceStable(
		n.element, func(i, j int) bool {
			return n.element[i].(metav1.Object).GetName() < n.element[j].(metav1.Object).GetName()
		},
	)
	return n
}

func (n *filter) toSlice() []interface{} {
	return n.element[0:]
}

//// isClusterAdmin judge weather is cluster scope kubeconfig or not
//func isClusterAdmin(clientset *kubernetes.Clientset) bool {
//	arg := &authorizationv1.SelfSubjectAccessReview{
//		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
//			ResourceAttributes: &authorizationv1.ResourceAttributes{
//				Namespace: "*",
//				Group:     "*",
//				Verb:      "*",
//				Name:      "*",
//				Version:   "*",
//				Resource:  "*",
//			},
//		},
//	}
//
//	response, err := clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(
//		context.TODO(), arg, metav1.CreateOptions{},
//	)
//	if err != nil || response == nil {
//		return false
//	}
//	return response.Status.Allowed
//}

// RemoveSearcherByKubeconfig remove informer from cache
func RemoveSearcherByKubeconfig(kubeconfigBytes []byte, namespace string) error {
	removeInformer(generateKey(kubeconfigBytes, namespace))
	c, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return err
	}
	list, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err == nil && list != nil {
		for _, item := range list.Items {
			removeInformer(generateKey(kubeconfigBytes, item.Namespace))
		}
	}
	return nil
}

func removeInformer(key string) {
	searchMapLock.Lock()
	defer searchMapLock.Unlock()
	if searcher, exist := searchMap.Get(key); exist && searcher != nil {
		searcher.(*Searcher).Stop()
		searchMap.Remove(key)
	}
}

// AddSearcherByKubeconfig init informer in advance
func AddSearcherByKubeconfig(kubeconfigBytes []byte, namespace string) error {
	searchMapLock.Lock()
	if searcher, exist := searchMap.Get(generateKey(kubeconfigBytes, namespace)); exist && searcher != nil {
		searchMapLock.Unlock()
		return nil
	}
	searchMapLock.Unlock()
	go func() { _, _ = GetSearcherWithLRU(kubeconfigBytes, namespace) }()
	return nil
}

func init() {
	go func() {
		for {
			select {
			case <-time.Tick(time.Minute * 5):
				go func() {
					defer func() {
						searchMapLock.Unlock()
						if err := recover(); err != nil {
							log.Warnf("check informer occurs error, err: %v", err)
						}
					}()
					searchMapLock.Lock()
					if searchMap != nil && searchMap.Len() > 0 {
						keys := searchMap.Keys()
						for _, key := range keys {
							if get, found := searchMap.Get(key); found && get != nil {
								if s, ok := get.(*Searcher); ok && s != nil {
									t := time.Time{}
									if s.lastUsedTime != t && time.Now().Sub(s.lastUsedTime).Hours() >= 24 {
										s.Stop()
										searchMap.Remove(key)
									}
								}
							}
						}
					}
				}()
			}
		}
	}()
}
