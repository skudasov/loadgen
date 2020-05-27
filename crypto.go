package loadgen

import (
	"encoding/pem"
	"github.com/insolar/x-crypto"
	"github.com/insolar/x-crypto/ecdsa"
	"github.com/insolar/x-crypto/x509"
)

// EncodeECDSAPair encodes private and public keys for reuse created members for later tests
func EncodeECDSAPair(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) (string, string) {
	x509Encoded, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		log.Fatal(err)
	}
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

	x509EncodedPub, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatal(err)
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	return string(pemEncoded), string(pemEncodedPub)
}

func DecodeECDSAPair(pemEncoded string, pemEncodedPub string) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	block, _ := pem.Decode([]byte(pemEncoded))
	x509Encoded := block.Bytes
	var privateKey crypto.PrivateKey
	privateKey, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		privateKey, err = x509.ParsePKCS8PrivateKey(x509Encoded)
		if err != nil {
			log.Fatal(err)
		}
	}

	blockPub, _ := pem.Decode([]byte(pemEncodedPub))
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509EncodedPub)
	if err != nil {
		log.Fatal(err)
	}
	return privateKey.(*ecdsa.PrivateKey), genericPublicKey.(*ecdsa.PublicKey)
}
