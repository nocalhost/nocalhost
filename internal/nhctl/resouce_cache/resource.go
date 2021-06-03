package resouce_cache

var GroupToTypeMap = []struct {
	K string
	V []string
}{
	{
		K: "Workloads",
		V: []string{
			"deployments",
			"statefulsets",
			"daemonsets",
			"jobs",
			"cronjobs",
			"pods",
		},
	},
	{
		K: "Networks",
		V: []string{
			"services",
			"endpoints",
			"ingresses",
			"networkpolicies",
		},
	},
	{
		K: "Configurations",
		V: []string{
			"configmaps",
			"secrets",
			"horizontalpodautoscalers",
			"resourcequotas",
			"poddisruptionbudgets",
		},
	},
	{
		K: "Storages",
		V: []string{
			"persistentvolumes",
			"persistentvolumeclaims", "storageclasses",
		},
	},
}
