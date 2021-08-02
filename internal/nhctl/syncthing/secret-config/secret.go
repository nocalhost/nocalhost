/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package secret_config

const (
	DefaultSyncthingHome       = "/var/syncthing"
	DefaultSyncthingSecretHome = "/var/syncthing/secret"
	SecretName                 = "nocalhost-syncthing-secret"
	EmptyDir                   = "nocalhost-syncthing"
)

// MDPJNTF-OSPJC65-LZNCQGD-3AWRUW6-BYJULSS-GOCA2TU-5DWWBNC-TKM4VQ5
const CertPEM = `-----BEGIN CERTIFICATE-----
MIIBsjCCATegAwIBAgIJAIGyJpDlj57UMAoGCCqGSM49BAMCMBQxEjAQBgNVBAMT
CXN5bmN0aGluZzAeFw0yMDEyMDIwMDAwMDBaFw00MDExMjcwMDAwMDBaMBQxEjAQ
BgNVBAMTCXN5bmN0aGluZzB2MBAGByqGSM49AgEGBSuBBAAiA2IABKwT2AZuXPkZ
qBhXvXmKzeLOhbLbm4Y7kzqC1sMPD8ZgDAeVigDkbQKdRUoQHa1ZclrI+KBWT5GB
TlfqSKB+P1S0XiXcOHpDZ5Hym6BIDwZKeEqellJDpCP0iAQuipUasKNVMFMwDgYD
VR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNV
HRMBAf8EAjAAMBQGA1UdEQQNMAuCCXN5bmN0aGluZzAKBggqhkjOPQQDAgNpADBm
AjEApJt+A90jZital6f4d/m9fOhk8fEqX4HwIrkC+BfhjPINBWrodkocsNkpyrQQ
htHzAjEAqHzP2veD9wJNXn9M9Pt7YJ1CvD4PZsxLYaNSVJzPVoeMUvbHlv+/RNko
/n/HhIOg
-----END CERTIFICATE-----
`

const KeyPEM = `-----BEGIN EC PRIVATE KEY-----
MIGkAgEBBDAfyZIpni7rVzW5/QL6jfS+8/0tuUyfDwzSdi3z0jFRtBGpMx5IEgCU
bOFBGe1ROvigBwYFK4EEACKhZANiAASsE9gGblz5GagYV715is3izoWy25uGO5M6
gtbDDw/GYAwHlYoA5G0CnUVKEB2tWXJayPigVk+RgU5X6kigfj9UtF4l3Dh6Q2eR
8pugSA8GSnhKnpZSQ6Qj9IgELoqVGrA=
-----END EC PRIVATE KEY-----
`
