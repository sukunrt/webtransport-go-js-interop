package main

import (
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
	fmt.Printf(
		`(async function main() {
    console.info("create session");
    const transport = new WebTransport("https://127.0.0.1:12345/say-hello", {
      serverCertificateHashes: [{
        algorithm: "sha-256",
        value: Uint8Array.from(atob("%s"), (m) => m.codePointAt(0))
      }]
    });

    console.info("wait for session");
    await transport.ready;
    console.info("session ready");

    const bds = transport.incomingBidirectionalStreams;
    const incomingStreamsReader = bds.getReader();

    // inbound loop
    (async () => {
      while (true) {
        const { done, value } = await incomingStreamsReader.read();
        if (done) {
          console.log("Connection lost");
          break;
        }
        console.info("received inbound stream");

        const reader = value.readable.getReader();
        try {
          const { value, done } = await reader.read();
          if (!done) {
            console.log("read from stream", value);
          }
        } catch (err) {
          console.info("error reading from stream", err);
        }

        console.log("close inbound stream");
        reader.cancel()
        value.writable.close();
      }
    })();

    // outbound loop
    (async () => {
      while (true) {
        console.info("create outbound stream");
        const stream = await transport.createBidirectionalStream();
        const writer = stream.writable.getWriter();
        await writer.ready;

        console.log("write to stream");
        try {
          const data = new Uint8Array([10, 10, 10]);
          writer.write(data);
          await writer.ready;
        } catch (err) {
          console.info("error writing to stream", err);
        }

        console.log("close outbound stream");
        stream.readable.cancel();
        writer.close()

        await new Promise(resolve => setTimeout(resolve, 1000));
      }
    })();
  })();`, certHash)
	fmt.Println()

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
		log.Println("incoming session")

		conn, err := s.Upgrade(w, r)
		if err != nil {
			log.Println("upgrading failed", err)
			w.WriteHeader(500)
			return
		}

		// inbound loop
		go func() {
			for {
				stream, err := conn.AcceptStream(conn.Context())
				if err != nil {
					log.Println("failed to accept stream", err)
				}
				log.Println("received inbound stream")

				buf := make([]byte, 15)
				read, err := stream.Read(buf)
				if err != nil {
					log.Println("error reading from stream", err)
				}

				log.Println("read from stream", buf[:read])

				log.Println("close inbound stream")
				stream.Close()
			}
		}()

		// outbound loop
		go func() {
			for {
				log.Println("create outbound stream")
				stream, err := conn.OpenStream()
				if err != nil {
					log.Println("failed to create stream", err)
				}

				log.Println("write to stream")
				n, err := stream.Write([]byte{10, 10, 10})
				if err != nil || n != 3 {
					log.Println("error writing to stream", err, n)
				}

				log.Println("close outbound stream")
				stream.Close()

				time.Sleep(time.Second)
			}
		}()
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
