// Package certparse extracts public certificate metadata without decoding private keys.
package certparse

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"ops-timer-backend/internal/model"
)

const MaxFileSize = 4 << 20

var ErrPrivateKey = errors.New("只接受公钥证书，不能上传私钥或包含私钥的文件")

func Parse(data []byte) ([]model.Certificate, error) {
	if len(data) == 0 || len(data) > MaxFileSize {
		return nil, errors.New("证书文件不能为空且不能超过 4 MiB")
	}
	trimmed := bytes.TrimSpace(data)
	if bytes.HasPrefix(trimmed, []byte("-----BEGIN PGP")) {
		if !bytes.HasPrefix(trimmed, []byte("-----BEGIN PGP PUBLIC KEY BLOCK-----")) {
			return nil, ErrPrivateKey
		}
		if bytes.Count(trimmed, []byte("-----BEGIN")) != 1 || !bytes.HasSuffix(trimmed, []byte("-----END PGP PUBLIC KEY BLOCK-----")) {
			return nil, errors.New("请上传单个完整的 GPG 公钥块")
		}
		block, err := armor.Decode(bytes.NewReader(trimmed))
		if err != nil || block.Type != openpgp.PublicKeyType {
			return nil, errors.New("GPG 公钥格式错误")
		}
		body, err := io.ReadAll(io.LimitReader(block.Body, MaxFileSize+1))
		if err != nil {
			return nil, errors.New("GPG 公钥数据或校验和错误")
		}
		return parseGPG(body)
	}
	if bytes.HasPrefix(trimmed, []byte("-----BEGIN")) {
		var ders [][]byte
		for len(trimmed) > 0 {
			if !bytes.HasPrefix(trimmed, []byte("-----BEGIN CERTIFICATE-----")) {
				if bytes.Contains(trimmed, []byte("PRIVATE KEY")) {
					return nil, ErrPrivateKey
				}
				return nil, errors.New("PEM 文件只能包含 CERTIFICATE 块")
			}
			endMarker := []byte("-----END CERTIFICATE-----")
			end := bytes.Index(trimmed, endMarker)
			if end < 0 {
				return nil, errors.New("PEM 证书不完整")
			}
			end += len(endMarker)
			segment := trimmed[:end]
			if bytes.Count(segment, []byte("-----BEGIN")) != 1 {
				return nil, errors.New("PEM 证书包含嵌套内容")
			}
			block, rest := pem.Decode(segment)
			if block == nil {
				return nil, errors.New("PEM 证书格式错误")
			}
			if len(bytes.TrimSpace(rest)) != 0 {
				return nil, errors.New("PEM 证书包含额外内容")
			}
			if strings.Contains(block.Type, "PRIVATE") {
				return nil, ErrPrivateKey
			}
			if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
				return nil, errors.New("PEM 文件只能包含 CERTIFICATE 块")
			}
			ders = append(ders, block.Bytes)
			trimmed = bytes.TrimSpace(trimmed[end:])
		}
		var result []model.Certificate
		for _, der := range ders {
			certs, err := parseX509(der)
			if err != nil {
				return nil, err
			}
			result = append(result, certs...)
		}
		return result, nil
	}
	if certs, err := parseX509(data); err == nil {
		return certs, nil
	}
	return parseGPG(data)
}

func parseX509(data []byte) ([]model.Certificate, error) {
	certs, err := x509.ParseCertificates(data)
	if err != nil || len(certs) == 0 {
		return nil, errors.New("X.509 证书格式错误")
	}
	result := make([]model.Certificate, 0, len(certs))
	for _, c := range certs {
		hash := sha256.Sum256(c.Raw)
		expires := c.NotAfter.UTC()
		result = append(result, model.Certificate{
			Type: "x509", CommonName: c.Subject.CommonName, Subject: c.Subject.String(), Issuer: c.Issuer.String(),
			SerialNumber: strings.ToUpper(c.SerialNumber.Text(16)), Fingerprint: strings.ToUpper(hex.EncodeToString(hash[:])),
			Algorithm: c.PublicKeyAlgorithm.String(), DNSNames: c.DNSNames, Emails: c.EmailAddresses,
			NotBefore: c.NotBefore.UTC(), ExpiresAt: &expires,
		})
	}
	return result, nil
}

func parseGPG(data []byte) ([]model.Certificate, error) {
	// Inspect packet tags before invoking the public-key decoder. Secret-key and
	// secret-subkey packet bodies are never parsed, including encrypted ones.
	opaque := packet.NewOpaqueReader(bytes.NewReader(data))
	primaryCount := 0
	for {
		p, err := opaque.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New("不是有效的 X.509 或 GPG 公钥证书")
		}
		if p.Tag == 5 || p.Tag == 7 {
			return nil, ErrPrivateKey
		}
		switch p.Tag {
		case 6:
			primaryCount++
		case 2, 10, 12, 13, 14, 17, 21:
		default:
			return nil, errors.New("GPG 文件包含非公钥证书数据")
		}
	}
	if primaryCount == 0 {
		return nil, errors.New("文件中没有 GPG 公钥证书")
	}
	reader := packet.NewReader(bytes.NewReader(data))
	result := make([]model.Certificate, 0, primaryCount)
	for i := 0; i < primaryCount; i++ {
		e, err := openpgp.ReadEntity(reader)
		if err != nil {
			return nil, fmt.Errorf("GPG 公钥证书解析失败: %w", err)
		}
		sig, identity := e.PrimarySelfSignature()
		if sig == nil {
			return nil, errors.New("GPG 公钥缺少有效的自签名")
		}
		if identity == nil {
			identity = e.PrimaryIdentity()
		}
		c := model.Certificate{Type: "gpg", Fingerprint: strings.ToUpper(hex.EncodeToString(e.PrimaryKey.Fingerprint)),
			SerialNumber: e.PrimaryKey.KeyIdString(), Algorithm: gpgAlgorithm(e.PrimaryKey.PubKeyAlgo),
			NotBefore: e.PrimaryKey.CreationTime.UTC(), Revoked: len(e.Revocations) > 0}
		if identity != nil {
			c.CommonName = identity.UserId.Name
			c.Subject = identity.UserId.Id
		}
		for uid, id := range e.Identities {
			c.UserIDs = append(c.UserIDs, uid)
			if id.UserId.Email != "" {
				c.Emails = append(c.Emails, id.UserId.Email)
			}
		}
		sort.Strings(c.UserIDs)
		sort.Strings(c.Emails)
		if sig.KeyLifetimeSecs != nil && *sig.KeyLifetimeSecs > 0 {
			expires := c.NotBefore.Add(time.Duration(*sig.KeyLifetimeSecs) * time.Second)
			c.ExpiresAt = &expires
		}
		result = append(result, c)
	}
	return result, nil
}

func gpgAlgorithm(a packet.PublicKeyAlgorithm) string {
	switch a {
	case packet.PubKeyAlgoRSA, packet.PubKeyAlgoRSAEncryptOnly, packet.PubKeyAlgoRSASignOnly:
		return "RSA"
	case packet.PubKeyAlgoDSA:
		return "DSA"
	case packet.PubKeyAlgoECDSA:
		return "ECDSA"
	case packet.PubKeyAlgoEdDSA:
		return "EdDSA"
	case packet.PubKeyAlgoEd25519:
		return "Ed25519"
	case packet.PubKeyAlgoEd448:
		return "Ed448"
	default:
		return fmt.Sprintf("OpenPGP algorithm %d", a)
	}
}
