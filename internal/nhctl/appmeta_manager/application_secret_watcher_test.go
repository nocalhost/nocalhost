package appmeta_manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ulikunitz/xz"
	"io"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nhctl/log"
	"os"
	"testing"
	"time"
)

var config = "apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJeE1EUXdPREF6TXpZME4xb1hEVE14TURRd05qQXpNelkwTjFvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTVR0CjY5YVBNVzVyaTdkaStEZFlRd2kwY0NCdVRmcjlnd0t3b2padlVGNkRYZ1BXdHNoRnE3NHhRbXR2UWdEYmxQMjEKMy9DUlA4UForZXpicGtJcEJ6SDhJVHNWQllUZWRsY1o5RU9LNnU5SUlhWit3d29YOEtzS0JtRW1scXdieWIrcgovUTlvc3dTQW5zSFdOR3JJa1ROaWNDNmIrQXBJMXpaNzVrNGN3aExuRHFCeHdRM1FRa2lNcFFIQzgyYVFBd2VaClcwbnFjNTk1M3BSMDA3M29OWXN1Ky8xd1BIbXVMQ3FxSjlPclJVb09IODFvMDd6cDJtNjZ2ak1raHg0dmlkRGkKeFJTaXJVQzZtT3JFbzNERERLTWV3bXRINHJLZUpDNWxMT1ZpMmswZUpwRk01UWh4RTJRWCtzYlJ2NnpWay9oawpDQ1hPYlMvSmFKcDU5ZHdKbVJVQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0tVTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFBU0xiaDhEcVRwVUhFN0dLRzVUWnRRQWVqUmcKZ1EvN0REQTNFcTJsSXRqM0Jvbm5yK0Q4MzMvejIrQkx0NGxBYnhUYVlBMjNLSFM4QWNKQm5qNHg1V1BXTzZRMwpNTlVjR3Q3MkJoQVhOR0VWV1JIbEwyZjFIaWZLTkFLdi84SHhxcm5qL3VrMjdyRGNSUFdpampYOWIxb1BucGxxClFJVm1PZ1RzVlpSTU9HYmRvd3lCOGtOelNoMTd3Z1FNV2xXNmVlK3BhUjI5QTY0YTZSbzd1Z3NidHVQL2xnenoKL0x5eW1aOHpuZE9VbFhwOWk4NjhnNlVQWTM4MkNpa3VoM2J0SnExM2ZDVU9tcTg3SWJ1Y0FpMFRwZ0h3RGpnZwpvak9KL3dkSGJaYnk3djV5RzQwL1ZnMlJIUm5CeXpKSXc5T2ZhK21LTG8vUE04ZDZ2cVppVWdVV01vST0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=\n    server: https://cls-p84thvx6.ccs.tencent-cloud.com\n  name: nh1xzyt\ncontexts:\n- context:\n    cluster: nh1xzyt\n    namespace: nh1xzyt\n    user: nocalhost-dev-account\n  name: nh1xzyt\ncurrent-context: nh1xzyt\nkind: Config\npreferences: {}\nusers:\n- name: nocalhost-dev-account\n  user:\n    token: eyJhbGciOiJSUzI1NiIsImtpZCI6Ijl1US13cnpTS3ZXcVprZnB0V1hnSWpQN3RRRVVfdE84MDM2S3gzZ2xJamsifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJuaDF4enl0Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZWNyZXQubmFtZSI6Im5vY2FsaG9zdC1kZXYtYWNjb3VudC10b2tlbi1nNm1tOSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJub2NhbGhvc3QtZGV2LWFjY291bnQiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC51aWQiOiIyYmU2NjVjNi00NzQ5LTQ1MTgtYThiMS01ZWZhZGQ4ZmU0OTkiLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6bmgxeHp5dDpub2NhbGhvc3QtZGV2LWFjY291bnQifQ.IWC1p3uJzJpsoZ_CGHYgus1ZM-SfbJayz4LDv6y1F7NouV5K-ZXJKUGpy6g2FUD3A80Mcnx98wj2a3MEnYiWXFV0hYDAu9M1jFhCLvEVcs8_gMCpQsfNzvcS6etNUSKtRBt28Af1NxHLWEXlDDO0d8TZSzC41chRk5HnaVRpceRBQF1EyOoRShlpzagi5vKjpz5WVFMACGduW1eoEEGNG4wS_K_4nCn_B1Mcglq1KyNEtJjbwflTJDAlqHETFxFHsXYBfp1iodPclhDhxNLdjfsChY_wbqQSMzSKK3oBwKTcCDl5HjTQes-B6KjbiK56189fBd01zhUMrfWg1ZhYvA"

func TestForWatch(t *testing.T) {
	Run()
	for i := 0; i < 10000; i++ {
		time.Sleep(time.Second)

		meta := GetApplicationMetas("nh1xzyt", config)
		marshal, _ := json.Marshal(meta)
		_ = fmt.Sprintf("meta: %s\n", string(marshal))

		uns := &appmeta.ApplicationMeta{}
		json.Unmarshal(marshal, uns)
		fmt.Sprintf("%s", string(marshal))
	}
}

func TestPackage(t *testing.T) {
	const text = "{\"configProperties\":{\"version\":\"v2\",\"envFile\":\"env.dev\"},\"application\":{\"name\":\"bookinfo\",\"manifestType\":\"rawManifest\",\"resourcePath\":[\"manifest/templates\"],\"ignoredPath\":[],\"onPreInstall\":[{\"path\":\"manifest/templates/pre-install/print-num-job-01.yaml\",\"weight\":\"1\"},{\"path\":\"manifest/templates/pre-install/print-num-job-02.yaml\",\"weight\":\"-5\"}],\"helmValues\":null,\"env\":[{\"name\":\"DEBUG\",\"value\":\"true\"},{\"name\":\"DOMAIN\",\"value\":\"coding.com\"}],\"envFrom\":{\"envFile\":null},\"services\":[{\"name\":\"productpage\",\"serviceType\":\"deployment\",\"dependLabelSelector\":{\"pods\":null,\"jobs\":[\"dep-job\"]},\"containers\":[{\"name\":\"productpage\",\"install\":{\"env\":null,\"envFrom\":{\"envFile\":null},\"portForward\":[\"39080:9080\"]},\"dev\":{\"gitUrl\":\"https://e.coding.net/codingcorp/nocalhost/bookinfo-productpage.git\",\"image\":\"codingcorp-docker.pkg.coding.net/nocalhost/dev-images/python:3.7.7-slim-productpage\",\"shell\":\"bash\",\"workDir\":\"/home/nocalhost-dev\",\"resources\":null,\"persistentVolumeDirs\":null,\"command\":null,\"debug\":null,\"useDevContainer\":false,\"sync\":{\"type\":\"send\",\"filePattern\":[\"./\"],\"ignoreFilePattern\":[\".git\",\".github\",\".idea\"]},\"env\":null,\"envFrom\":null,\"portForward\":[\"39080:9080\"]}}]},{\"name\":\"details\",\"serviceType\":\"deployment\",\"dependLabelSelector\":null,\"containers\":[{\"name\":\"\",\"install\":null,\"dev\":{\"gitUrl\":\"https://e.coding.net/codingcorp/nocalhost/bookinfo-details.git\",\"image\":\"codingcorp-docker.pkg.coding.net/nocalhost/dev-images/ruby:2.7.1-slim\",\"shell\":\"bash\",\"workDir\":\"/home/nocalhost-dev\",\"resources\":null,\"persistentVolumeDirs\":null,\"command\":null,\"debug\":null,\"useDevContainer\":false,\"sync\":{\"type\":\"send\",\"filePattern\":[\"./\"],\"ignoreFilePattern\":[\".git\",\".github\"]},\"env\":[{\"name\":\"DEBUG\",\"value\":\"true\"}],\"envFrom\":null,\"portForward\":null}}]},{\"name\":\"ratings\",\"serviceType\":\"deployment\",\"dependLabelSelector\":{\"pods\":[\"productpage\",\"app.kubernetes.io/name=productpage\"],\"jobs\":[\"dep-job\"]},\"containers\":[{\"name\":\"\",\"install\":null,\"dev\":{\"gitUrl\":\"https://e.coding.net/codingcorp/nocalhost/bookinfo-ratings.git\",\"image\":\"codingcorp-docker.pkg.coding.net/nocalhost/dev-images/node:12.18.1-slim\",\"shell\":\"bash\",\"workDir\":\"/home/nocalhost-dev\",\"resources\":null,\"persistentVolumeDirs\":null,\"command\":null,\"debug\":null,\"useDevContainer\":false,\"sync\":{\"type\":\"send\",\"filePattern\":[\"./\"],\"ignoreFilePattern\":[\".git\",\".github\",\"node_modules\"]},\"env\":[{\"name\":\"DEBUG\",\"value\":\"true\"}],\"envFrom\":null,\"portForward\":null}}]},{\"name\":\"reviews\",\"serviceType\":\"deployment\",\"dependLabelSelector\":{\"pods\":[\"productpage\"],\"jobs\":null},\"containers\":[{\"name\":\"\",\"install\":null,\"dev\":{\"gitUrl\":\"https://e.coding.net/codingcorp/nocalhost/bookinfo-reviews.git\",\"image\":\"codingcorp-docker.pkg.coding.net/nocalhost/dev-images/java:latest\",\"shell\":\"bash\",\"workDir\":\"/home/nocalhost-dev\",\"resources\":null,\"persistentVolumeDirs\":null,\"command\":null,\"debug\":null,\"useDevContainer\":false,\"sync\":{\"type\":\"send\",\"filePattern\":[\"./\"],\"ignoreFilePattern\":[\".git\",\".github\",\".gradle\",\"build\"]},\"env\":null,\"envFrom\":null,\"portForward\":null}}]}]}}"
	var buf bytes.Buffer
	// compress text
	w, err := xz.NewWriter(&buf)
	if err != nil {
		log.Fatalf("xz.NewWriter error %s", err)
	}
	if _, err := io.WriteString(w, text); err != nil {
		log.Fatalf("WriteString error %s", err)
	}
	if err := w.Close(); err != nil {
		log.Fatalf("w.Close error %s", err)
	}

	// decompress buffer and write output to stdout
	r, err := xz.NewReader(&buf)
	if err != nil {
		log.Fatalf("NewReader error %s", err)
	}
	if _, err = io.Copy(os.Stdout, r); err != nil {
		log.Fatalf("io.Copy error %s", err)
	}
}
