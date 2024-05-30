package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	b64 "encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/quic-go/quic-go/http3"

	"github.com/quic-go/webtransport-go"
)

func main() {
	tlsConf, err := getTLSConf(time.Now(), time.Now().Add(10*24*time.Hour))
	if err != nil {
		log.Fatal(err)
	}
	hash := sha256.Sum256(tlsConf.Certificates[0].Leaf.Raw)

	// print the certhash
	certHash := b64.StdEncoding.EncodeToString(hash[:])
	fmt.Println(certHash)

	wmux := http.NewServeMux()
	s := webtransport.Server{
		H3: http3.Server{
			TLSConfig: tlsConf,
			Addr:      "localhost:12345",
			Handler:   wmux,
		},
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	defer s.Close()

	wmux.HandleFunc("/say-hello", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("SERVER incoming session")

		conn, err := s.Upgrade(w, r)
		if err != nil {
			log.Printf("SERVER upgrading failed: %s", err)
			w.WriteHeader(500)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("SERVER wait for stream")
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			log.Fatalf("SERVER failed to accept bidirectional stream: %v", err)
		}

		defer stream.Close()

		log.Printf("SERVER says hello")

		for i := 0; i < 256; i++ {
			if _, err = stream.Write(make([]byte, 1024 * 1024)); err != nil {
				log.Fatal(err)
			}
		}

		log.Printf("SERVER stream finished")
	})

	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func getTLSConf(start, end time.Time) (*tls.Config, error) {
	cert, priv, err := generateCert(start, end)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{cert.Raw},
			PrivateKey:  priv,
			Leaf:        cert,
		}},
	}, nil
}

func generateCert(start, end time.Time) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return nil, nil, err
	}
	serial := int64(binary.BigEndian.Uint64(b))
	if serial < 0 {
		serial = -serial
	}
	certTempl := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{},
		NotBefore:             start,
		NotAfter:              end,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, certTempl, certTempl, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, nil, err
	}
	ca, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return nil, nil, err
	}
	return ca, caPrivateKey, nil
}
