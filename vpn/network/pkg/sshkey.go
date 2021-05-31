package pkg

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type sshInfo struct {
	PrivateKeyBytes []byte
	PublicKeyBytes  []byte
	PrivateKeyPath  string
}

func generateSshKey(privateKeyPath string) (*sshInfo, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if err = privateKey.Validate(); err != nil {
		log.Println(err)
		return nil, err
	}
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	privateKeyBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	info := sshInfo{
		PublicKeyBytes:  publicKeyBytes,
		PrivateKeyBytes: privateKeyBytes,
		PrivateKeyPath:  privateKeyPath,
	}
	if err = saveKeyToDisk(privateKeyPath, privateKeyBytes); err != nil {
		log.Printf("write private key failed, error: %v\n", err)
		return &info, err
	}
	return &info, nil
}

func saveKeyToDisk(keyPath string, data []byte) error {
	dir := filepath.Dir(keyPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0700); err != nil {
			log.Println("can't create dir")
			return err
		}
	}
	if err := ioutil.WriteFile(keyPath, data, 0400); err != nil {
		log.Println("write ssh private key failed")
		return err
	}
	return nil
}

func HomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home
	}
	return "/root"
}
