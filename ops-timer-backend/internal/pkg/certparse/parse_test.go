package certparse

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

func TestX509FormatsAndPrivateKeyRejection(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(42), Subject: pkix.Name{CommonName: "example.com"}, NotBefore: start, NotAfter: start.Add(24 * time.Hour), DNSNames: []string{"example.com", "www.example.com"}, EmailAddresses: []string{"ops@example.com"}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	public := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	for _, data := range [][]byte{der, public, append(append([]byte{}, public...), public...)} {
		certs, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range certs {
			if c.CommonName != "example.com" || c.ExpiresAt == nil || !c.ExpiresAt.Equal(tmpl.NotAfter) || !c.NotBefore.Equal(start) || c.SerialNumber != "2A" || len(c.Fingerprint) != 64 || len(c.DNSNames) != 2 || len(c.Emails) != 1 {
				t.Fatalf("unexpected metadata: %+v", c)
			}
		}
	}
	privateDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	private := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateDER})
	for _, data := range [][]byte{private, append(append([]byte{}, public...), private...), append(append([]byte{}, private...), public...)} {
		if _, err := Parse(data); !errors.Is(err, ErrPrivateKey) {
			t.Fatalf("private material accepted: %v", err)
		}
	}
	for _, data := range [][]byte{privateDER, append(der, 0), []byte("garbage"), []byte{}, append(append([]byte{}, public...), []byte("garbage")...), append([]byte("-----BEGIN CERTIFICATE-----\nbad\n"), public...), bytes.Repeat([]byte("a"), MaxFileSize+1)} {
		if _, err := Parse(data); err == nil {
			t.Fatal("invalid file accepted")
		}
	}
}

func TestGPGExpiryAndPrivatePacketRejection(t *testing.T) {
	created := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	for _, lifetime := range []uint32{0, 7200} {
		for _, v6 := range []bool{false, true} {
			cfg := &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA, Curve: packet.Curve25519, Time: func() time.Time { return created }, KeyLifetimeSecs: lifetime, V6Keys: v6}
			if v6 {
				cfg.Algorithm = packet.PubKeyAlgoEd25519
			}
			e, err := openpgp.NewEntity("Alice", "Ops", "alice@example.com", cfg)
			if err != nil {
				t.Fatal(err)
			}
			var public, secret bytes.Buffer
			if err := e.Serialize(&public); err != nil {
				t.Fatal(err)
			}
			if err := e.SerializePrivate(&secret, cfg); err != nil {
				t.Fatal(err)
			}
			var armored bytes.Buffer
			w, err := armor.Encode(&armored, openpgp.PublicKeyType, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(public.Bytes()); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			for _, data := range [][]byte{public.Bytes(), armored.Bytes()} {
				certs, err := Parse(data)
				if err != nil || len(certs) != 1 {
					t.Fatalf("parse v6=%v: %v", v6, err)
				}
				c := certs[0]
				if c.Type != "gpg" || c.CommonName != "Alice" || !c.NotBefore.Equal(created) || len(c.UserIDs) != 1 || c.Emails[0] != "alice@example.com" || c.Fingerprint == "" {
					t.Fatalf("metadata: %+v", c)
				}
				if lifetime == 0 && c.ExpiresAt != nil {
					t.Fatal("unlimited key has expiry")
				}
				if lifetime != 0 && (c.ExpiresAt == nil || !c.ExpiresAt.Equal(created.Add(time.Duration(lifetime)*time.Second))) {
					t.Fatal("wrong GPG expiry")
				}
			}
			for _, data := range [][]byte{secret.Bytes(), append(append([]byte{}, public.Bytes()...), secret.Bytes()...)} {
				if _, err := Parse(data); !errors.Is(err, ErrPrivateKey) {
					t.Fatalf("secret packets accepted: %v", err)
				}
			}
			// Reject the secret-subkey tag before attempting to decode its body.
			withSecretSubkey := append(append([]byte{}, public.Bytes()...), 0xc7, 1, 0)
			if _, err := Parse(withSecretSubkey); !errors.Is(err, ErrPrivateKey) {
				t.Fatalf("secret subkey decoded: %v", err)
			}
			if _, err := Parse(append(append([]byte{}, armored.Bytes()...), []byte("extra")...)); err == nil {
				t.Fatal("armored trailing data accepted")
			}
		}
	}
}
