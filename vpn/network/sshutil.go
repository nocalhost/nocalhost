package network

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

type SSHInfo struct {
	PrivateKeyBytes []byte
	PublicKeyBytes  []byte
}

func generateSSH(privateKeyPath string, publicKeyPath string) *SSHInfo {
	privateKey, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		log.Println(err)
	}
	if err = privateKey.Validate(); err != nil {
		log.Println(err)
	}
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Println(err)
	}
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	privateKeyBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	err = saveKeyToDisk(privateKeyPath, privateKeyBytes)
	err = saveKeyToDisk(publicKeyPath, publicKeyBytes)
	if err != nil {
		log.Println(err)
	}
	return &SSHInfo{PublicKeyBytes: publicKeyBytes, PrivateKeyBytes: privateKeyBytes}
}

func saveKeyToDisk(privateKeyPath string, data []byte) error {
	dir := filepath.Dir(privateKeyPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0700); err != nil {
			log.Println("can't create dir")
			return err
		}
	}
	if err := ioutil.WriteFile(privateKeyPath, data, 0400); err != nil {
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

func GenerateKey(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	return private, &private.PublicKey, nil
}

func EncodePrivateKey(private *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Bytes: x509.MarshalPKCS1PrivateKey(private),
		Type:  "RSA PRIVATE KEY",
	})
}

func EncodePublicKey(public *rsa.PublicKey) ([]byte, error) {
	publicBytes, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Bytes: publicBytes,
		Type:  "PUBLIC KEY",
	}), nil
}

func EncodeSSHKey(public *rsa.PublicKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(public)
	if err != nil {
		return nil, err
	}
	return ssh.MarshalAuthorizedKey(publicKey), nil
}

func MakeSSHKeyPair() (string, string, error) {

	pkey, pubkey, err := GenerateKey(2048)
	if err != nil {
		return "", "", err
	}

	pub, err := EncodeSSHKey(pubkey)
	if err != nil {
		return "", "", err
	}
	return string(EncodePrivateKey(pkey)), string(pub), nil
}
