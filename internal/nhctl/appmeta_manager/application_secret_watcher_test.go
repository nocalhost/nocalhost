package appmeta_manager

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

var config = "apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJeE1ESXlOREEyTlRZeE9Gb1hEVE14TURJeU1qQTJOVFl4T0Zvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTUFRCkVoL0JqZFIwSmFnTldLTmJrUWtXSUh5eGVEck1rczlmK3dUNUFST3k1bmdsa1UxeHNUZEFqRkdLUDlmWnROSW8KR2hwajdrYmJWWFA5TVp4cWZHeVFoR1R1TEpTbEhaMXVUMy83NTVUY1FudmZydmJ3M2NTWU8zNTZaMEh0YmZQRApKcFdaSFdGVERtenZjY0w0dzVVZytYdUZHRG9QcU9vMHpycTZnWGxqZ1pZZ3E4a0VaemFPYW5lVG9wTmtzSVNsCk5QeE0wYWtoY0NCakRIaVg1VzUyQ295ME4xL0RSbVZzV0NqcWRLSW0zWWp0cVNWUFlidlpCUUdVcUVWL3lONk0KVGF4cVV2Ny8xaXpxMXBDZGlRZEx6dnIvT1BlVjBEOGxUWGptcE1tYVdQVmtLQ2s2VzR3QUlKNEQzUmRiZGxIMwoycE9VLzFycis1ZWFzYXUvaWhFQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0tVTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFHZk1naTY5ZDFyYm14Rkx1TnZRZkxwa2VNZnYKWTJnTGxLck1KZDdNeW52ZFVVTXV3NFkxTG83MkxkU3pqaU9lbkxhazN0MWhBR0ZJQ3RNdGpkUm1JZ1p1YWJLMApKN3hQLzdtcHJRdmtoWit4WlJkaVA2ZkkrQzVKSlBSRXFnbEtxWjkybWg4dEdPa1FBSFBsVWo2dVFvRmZxMkpBCkZxc29jaXRteUFRMzJvOUN1eWltTnNjZmIvR24yck1EdTMvbXVWTjFBYkJNeFhnTm4yWFZMakJEMHFwbGVXVDMKS244VWk2U1N2TWcyd1JHWjBER1NCK1hWN1ZBc1FYdEtqNGtlaWE4cmdrRmxsTGJJWGdSdE1oQjFjQTJaRjdmSgp5SXIrQXlYaStiSWxrOHA3Q3Q0Uk0renJjU2lpQnh2ZHhuVGsyUnljdmxac0VudVBSQ21tRXVhcmlVST0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=\n    server: https://cls-machfc76.ccs.tencent-cloud.com\n  name: cls-machfc76\ncontexts:\n- context:\n    cluster: cls-machfc76\n    user: \"100013417882\"\n  name: cls-machfc76-100013417882-context-default\ncurrent-context: cls-machfc76-100013417882-context-default\nkind: Config\npreferences: {}\nusers:\n- name: \"100013417882\"\n  user:\n    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURERENDQWZTZ0F3SUJBZ0lJWE1JdXJQRjlyQzB3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TVRBeU1qUXdOalU0TURaYUZ3MDBNVEF5TWpRd05qVTRNRFphTURZeApFakFRQmdOVkJBb1RDWFJyWlRwMWMyVnljekVnTUI0R0ExVUVBeE1YTVRBd01ERXpOREUzT0RneUxURTJNVFF4Ck5EazRPRFl3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLQW9JQkFRRFhHY0JYSzJuWDh6MkIKMm5MUW9qNE5kYk5BWFZmK1QyOGo0NmpaVmR0dDJ1NTFWOUU1c1dFanoyWmltZTBmdzRLRnNzWUtSemF0bXdQTQprTnQwR29KN1AzVjlTZXJkbTVyOGp0WW4vUzBWRGJMcTU2KzBvUGhMV3pkZGx2T0QzK0gydDNsZGtOMlA1dTE5ClFkeFFRTUJ1NHkzZks2LzZ5YjR4ckNGQXJ4d3lxU0dvQUNzak8yUUdZYkdpRGdOT2c0bTVoUTZQRUF4R05uK3oKNkdnRTdqS25ZeWppa2pHVk5OS3JWcCtuZTlleE41ekI4N1I2UnhFeVE4a21GWTA2VFp1eXJJbDIvQ2t3QWlzcwpKY1lJcU9xaVRScUlSZXNyNVIvNmd4RXVVcFhDako4eE1HN1h6eGVtcEdvOStaV3d1ZmVXOER5M1NWMUdpTDZLCk5BeDFZSy9mQWdNQkFBR2pQekE5TUE0R0ExVWREd0VCL3dRRUF3SUNoREFkQmdOVkhTVUVGakFVQmdnckJnRUYKQlFjREFnWUlLd1lCQlFVSEF3RXdEQVlEVlIwVEFRSC9CQUl3QURBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQQpEcUVWSlRiZzhBbDFCSWM0MUdqWG5nbkxRTDV5ZHZyOGlxcXZvZ2JmVkxNT2U0VmZKaGZtWHQ4b0ZFdnJwdlRnCmxLMzNYc3pWbW02Z1liWDlSbzZpYS9ZRkE0UVMwaUltMnBxSXJNcW9iM3ZINnZkajBhOTFXZFRrZzdVMHpXcWcKc090UkpvVG1BV3N2WWlXZWwxS3dIMUlROEQwb0s2UjNNa25CczJzSGRMVm9BTmFmV3hPMnVyZ1BuVklpQ2xCNwppcGpjNlJobGtFcCtqdU1pL09rN1JER2t4aFhYNFI4dTJJNW9qZGhQeHpqWE04UkJ0Y2NmbHplYnBib2V5eGVVCjBOTE1TTVptTVpzSkcxRXl3UnpwaUQxODJrTmFDbllBQ0NuZE91ZE9JWEpUVjBWOXVTK3E2VFhpeDFRYml0ZUkKN2kwRm0zakVoVWNtOHdEN2hkQzdCQT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBMXhuQVZ5dHAxL005Z2RweTBLSStEWFd6UUYxWC9rOXZJK09vMlZYYmJkcnVkVmZSCk9iRmhJODltWXBudEg4T0NoYkxHQ2tjMnJac0R6SkRiZEJxQ2V6OTFmVW5xM1p1YS9JN1dKLzB0RlEyeTZ1ZXYKdEtENFMxczNYWmJ6ZzkvaDlyZDVYWkRkaitidGZVSGNVRURBYnVNdDN5dXYrc20rTWF3aFFLOGNNcWtocUFBcgpJenRrQm1HeG9nNERUb09KdVlVT2p4QU1Salovcytob0JPNHlwMk1vNHBJeGxUVFNxMWFmcDN2WHNUZWN3Zk8wCmVrY1JNa1BKSmhXTk9rMmJzcXlKZHZ3cE1BSXJMQ1hHQ0tqcW9rMGFpRVhySytVZitvTVJMbEtWd295Zk1UQnUKMTg4WHBxUnFQZm1Wc0xuM2x2QTh0MGxkUm9pK2lqUU1kV0N2M3dJREFRQUJBb0lCQVFDR0hoRldpTVFyR1FndAowaVlxdmk2UXQrVzNhVHczWGhIL1A3RUZLa3B5T2NMYk9aRkVOcnhKMXNTUkVFYlF1bGZFdzA1R0ZGY2NjZjR4CmE5VFpsTG5zM1FtRndEUUlUMENZM3ZyYTNqcGcyVFRJMFFNMlRmUGpFSkg1OGVnT1B1Y21yUW1vZEc0aGpxeGYKb0ZRZFdmSmljWllsZzVqcmR5VDIxY3U2Q0RVOXhDQjU4Q2gwNGQ5cDRWaVVaTTBXVDE5cFZ3eDl3RnpQcjdhZApXUHFXbVE2VzlEdUZMUC8xQ2VmVmRmNXJ3UEpaZGFVK01QdTlSZzBnS1Y5RVkvdnFEalFITGo3VlRLaTdsOGN6Cjd2RTZkSlBRODNkWXpSMWlVSEtyU1VtUmdqd1RhK1NxQ1NvSVNDYyttVWorMVhLM0Ryb3o1QkxIRlgxM2x2d3UKbG1mdk5ueEJBb0dCQVBBVUg0Z3RCV1A2OURDV05wQ09QY0U5NmVtQXVSWEtuaSt0aGloUEJxWUNpL3cyUUlmeAp6d2ZZVnpHcVYzU0N2c2pLbW0ydm5LL2hQN0grUHduU1k5dTUrdVBhdkcybEkwc0V6S1U2MzZsWkJCNks5U200CmE0SjB2UTQ0SXFPRDQ1bm02b2VBQnlYaVNaZTA3RFdMSlpnWi9obXBHaDA2WnJmRUJEWjBmeEloQW9HQkFPVmQKa2VYTGRzSUgwN1NyamhDeGNuV2RCcXRRRFc5b0JzOVZFUUFvRFZHa1FMUGtDcHpBVXI5N2lSVFVTd2pZcUFLbApnbi9rMzJjNURsNU04d0tZS0drR2taNzQ4L0pZZmVwL3JuS3JsTlJFdjJHdzhiZUpsQldZbCtRSWtET3FFakFjCldMeEFKSVEvSjU5WU9rUVZXMjRtY2czdmgyL1VMQzFMQmRPelM0SC9Bb0dBV1BON3cxNjY4cEpXeTNHOGdjN1IKL3JsTDQ2STM4V1VET3pNVjAvV0R4eHFHZDBvNm1xUHpTenJUQTZuVGdXMjM5bmxxd2wwZ3R1SEVVZFNiMHEzTApKZXhBa3cvR1pQR2NvLzBCUGU4VVU1Q1J3Q2RJTXM4THRtZythL2hNalQwZXBUVXpqRVRaWVNYNGttY01aY0pLCmlaS0gzVVlVVU9RRWp1M25pYTJjTDBFQ2dZRUE0Y0svQjV2RVVlbVlWUjREWUtUNGo1RzI3Y3FHM3VCYXk1cmsKZCszMFppYXhWUitoM25aalBIeWhDaktIaExhVWNMNXVlK3BRaHU2ZkdPek95UC94enFhYmtRbGtQR2NqMFR4SgovaTZxK0dDT3E5NlpuVms2dkNNTlpuT1RWSGNUSGUzWTNicVk5dDZlNW5YV0xBdUZpaDhuWmxZZFRsSmVCVnJ4CjZsVnhmZ3NDZ1lCWTNmVmUvVk5RV3JQRENacjRMcW5kRURYV3NoL0xSenJlZ3ZkVjdXTFFtb2xVYnE5S0tzRVkKYjBZOGRSays0UktGUDliSWpTUFFvQmJOVi9jemxrZmJYMTRlNHo5M09IZnFkcVI3VVlmbkkwQTRXNVozUzA4MAoveUZGTHlRV2JBM0t3V2NaU0NYWmp2bHNFT2lkblQ3aS8zWnNnMHhReCsweko4YTZ6Nm8wbWc9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo="

func TestForWatch(t *testing.T) {
	Run()
	for i := 0; i < 10000; i++ {
		time.Sleep(time.Second)


		meta := GetApplicationMeta("nh2junh", "bookinfo", config)
		marshal, _ := json.Marshal(meta)
		_ = fmt.Sprintf("meta: %s\n", string(marshal))
		//
		//uns := &appmeta.ApplicationMeta{}
		//json.Unmarshal(marshal, uns)
		//fmt.Sprintf("")
	}
}
