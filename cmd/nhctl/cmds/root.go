package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"nocalhost/pkg/nhctl/utils"
	"os"
	"strings"
)

var settings *EnvSettings

var nocalhost = &NocalHost{}

func init() {

	settings = NewEnvSettings()

	rootCmd.PersistentFlags().BoolVar(&settings.Debug, "debug", settings.Debug, "enable debug level log")
	rootCmd.PersistentFlags().StringVar(&settings.KubeConfig, "kubeconfig", "", "the path to the kubeconfig file")

	cobra.OnInitialize(func() {
		var (
			nhctlHomeDirName = ".nhctl"
		)
		nhctlHomeDir := fmt.Sprintf("%s%c%s", GetHomePath(), os.PathSeparator, nhctlHomeDirName)
		if _, err := os.Stat(nhctlHomeDir); err != nil {
			if os.IsNotExist(err) {
				debug("init nhctl...")
				utils.Mush(os.Mkdir(nhctlHomeDir, 0755))
				applicationDir := fmt.Sprintf("%s%c%s", nhctlHomeDir, os.PathSeparator, "application")
				utils.Mush(os.Mkdir(applicationDir, 0755)) // create .nhctl/application
				keyDir := fmt.Sprintf("%s%c%s", nhctlHomeDir, os.PathSeparator, "key")
				utils.Mush(os.Mkdir(keyDir, 0755)) // create .nhctl/key
				// ssh public key
				keyContent := `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDqJOIfjQvv2pAanw3PBjpIqda+F7QAY0C4818D76C4u5Ybrja+Fz0cOCjtrRuwopsNcZhbGrva/zuG8J7Violft294fYVils7gOi1FjzA2twU1n90nCFpHt5uxETR9jR7JpsTUq15Xi6aIB5PynF/irr3EueUiiywhvzejbr1sA0ri26wteaSr/nLdNFy2TXVAEyHyzoxCAX4cECuGfarIgoQpdErc6dwyCh+lPnByL+AGP+PKsQmHmA/3NUUJGsurEf4vGaCd0d7/FGtvMG+N28C33Rv1nZi4RzWbG/TGlFleuvO8QV1zqIGQbUkqoeoLbbYsOW2GG0BxhJ7jqj9V root@eafa293b895`
				publicKeyFile := fmt.Sprintf("%s%c%s", keyDir, os.PathSeparator, "id_rsa.pub")

				utils.Mush(ioutil.WriteFile(publicKeyFile, []byte(keyContent), 0644))

				privateKeyContent := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA6iTiH40L79qQGp8NzwY6SKnWvhe0AGNAuPNfA++guLuWG642
vhc9HDgo7a0bsKKbDXGYWxq72v87hvCe1YqJX7dveH2FYpbO4DotRY8wNrcFNZ/d
JwhaR7ebsRE0fY0eyabE1KteV4umiAeT8pxf4q69xLnlIossIb83o269bANK4tus
LXmkq/5y3TRctk11QBMh8s6MQgF+HBArhn2qyIKEKXRK3OncMgofpT5wci/gBj/j
yrEJh5gP9zVFCRrLqxH+LxmgndHe/xRrbzBvjdvAt90b9Z2YuEc1mxv0xpRZXrrz
vEFdc6iBkG1JKqHqC222LDlthhtAcYSe46o/VQIDAQABAoIBAQCUUhb325ZjMzWz
12uc6BoFq6i/tC4vTLBUOL7ItIRAYXwePsaotfndJWov3Ue8JdVIt9vGYnH7sVDZ
ExXaua559q5jSkgzgsq72b6R4Lmu/1MKfCFQt4bRBWtXyElS+xE0tjLbcU8K8Ajn
BL3gotROuVi3BPc0YarsGcA6BE1z3I+d2dY1LI59r8KiHcgwbRFMfoSYfk20LaWr
NJqa4wSIS8h4R8121o+1W7zyX+z3WAfEDCcCcy+0ewj+REUdzeDpHhI9AP9itIGI
fFkb4O3J8sYoyX3aKl4tRPbvrJAX3wntei4Gr8E304TCcqfvr4waw+xSf+gQEtUy
NIkhlo9lAoGBAP98yRaeP2mQOuU7NalhtANDhqAcW8zPyKL87ruUDls9gCoL/mAD
Qty+8m74hmUfMtQBHJobiNlziNi4l2WeFw2S1/Z34J28Vb8cgopXjyQHLY/6//3r
vLqL74fEru9tRqV0pmY4YMD4CX/0DR0QNInYNeI7mPWip4ZmhcdiyGHfAoGBAOqd
IttrSOJqOvh97MEXupRLDIDoD2rh+wg3vT1o8zbuR+YMwpxtdKn4hoe11W5xJHS+
jRESkKcRckuatf/qLEUrsCcCjPC6hqMKCsAR0Jsl42/dyd4v6WNFouoipW7XewPb
8jVwzLzF4p+5kjw7iCuXmfqKDxWwHfZOsT9SKs1LAoGAXJ4SD98CQfSFRUB3rZW7
uksqbLSbGt5gb6WdreZ4Zd8frR538rp77KZUIKJ7pgDvXiehBMTikWHuxBH24GG1
HbiUDcdbaBM0Snm9YQVo4LixbbaiQpzI6B9+kAtfF3DX4XcuM3RQruO8HeSNNHIB
ec8liYPtaW6zqGdWK/fFiKUCgYBRq83ckCZZGx3YLw3h0f7TbKS3oxDq5ivbGnw4
CnbQInbI8Jw2lCvOl4NNbtETlzNXqJW24b2VSw98nijJI52xnpm9mrexfV0tGGvR
nOH/gFsCMDT7sbYPJsiltNXeFgjuuPxB+jhrZn+Tlqf/a8HlWurxOmox5JMpkQ9G
ubXIrQKBgQCXgCKqDuA3pqiupRWZ3K2VA19FkGXfikM/uS6F26rhiPaoebhfsM3T
oMu7FrusZvhZhbqEhRMIJ1+HlqPsYdFlDHmJ3tztS5cG8+XMwOaLQOpbof2WEoJa
4GlJO+705cY37lnlLNb2bEaPhNFnHMkTSfFYmsOm4qWab4fubRTrFQ==
-----END RSA PRIVATE KEY-----`

				privaetKeyFile := fmt.Sprintf("%s%c%s", keyDir, os.PathSeparator, "id_rsa")
				utils.Mush(ioutil.WriteFile(privaetKeyFile, []byte(privateKeyContent), 0600))
			}
		}

		sshHomeDir := fmt.Sprintf("%s%c%s", GetHomePath(), os.PathSeparator, ".ssh")
		sshConfigFile := fmt.Sprintf("%s%c%s", sshHomeDir, os.PathSeparator, "config")
		sshConfig := `
Host shared-container
HostName 127.0.0.1
User root
IdentityFile ~/.nhctl/key/id_rsa
`

		if _, err := os.Stat(sshHomeDir); err != nil {
			if os.IsNotExist(err) {
				debug(".ssh not exists, create it")
				os.Mkdir(sshHomeDir, 0644)
			} else {
				panic(err)
			}
		}

		if _, err := os.Stat(sshConfigFile); err != nil {
			if os.IsNotExist(err) {
				debug("~/.ssh/config not exists, create it")
				utils.Mush(ioutil.WriteFile(sshConfigFile, []byte(""), 0644))
			} else {
				panic(err)
			}
		}

		f, err := ioutil.ReadFile(sshConfigFile)
		utils.Mush(err)

		if strings.Contains(string(f), "Host shared-container") {
			debug("~/.ssh/config already config, ignore it")
			return
		}

		file, err := os.OpenFile(sshConfigFile, os.O_WRONLY|os.O_APPEND, 0644)
		utils.Mush(err)
		defer file.Close()

		_, err = file.Write([]byte(sshConfig))
		utils.Mush(err)
	})
}

var rootCmd = &cobra.Command{
	Use:   "nhctl",
	Short: "nhctl use to deploy coding project",
	Long:  `nhctl can deploy project on Kubernetes. `,
	Run: func(cmd *cobra.Command, args []string) {
		debug("hello nhctl")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
