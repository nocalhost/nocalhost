package kubectl

// should be run in the background ??
//func PortForward(ctx context.Context, kubeconfig string, namespace string, deployment string, localPort string, remotePort string) error {
//	fmt.Println("kubectl port-forwarding...")
//	_, active := tools.CheckK8s()
//	if !active {
//		fmt.Println("kubernetes cluster is unavailable")
//		return errors.New("kubernetes cluster is unavailable")
//	}
//	return tools.ExecKubeCtlCommand(ctx, kubeconfig, "port-forward", "--address", "0.0.0.0", fmt.Sprintf("deployment/%s", deployment), fmt.Sprintf("%s:%s", localPort, remotePort), "-n", namespace)
//}
