package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/service"
)

func TestPostgresCertificateAttachments(t *testing.T) {
	db := testPostgres(t)
	svc := service.NewCertificateService(db)
	h := handler.NewCertificateHandler(svc)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/attachments", h.Upload)
	r.GET("/attachments", h.List)
	r.GET("/attachments/:id/download", h.Download)
	r.DELETE("/attachments/:id", h.Delete)
	r.POST("/attachments/:id/links", h.Link)
	r.DELETE("/attachments/:id/links", h.Link)
	r.POST("/certificates/:id/countdown", h.CreateCountdown)
	request := func(method, path, body string, status int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s: %d: %s", method, path, w.Code, w.Body.String())
		}
		return w.Body.Bytes()
	}
	project := model.Project{ID: uuid.NewString(), Title: "Certificate project"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	expires := time.Now().UTC().Truncate(time.Second).Add(24 * time.Hour)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(99), Subject: pkix.Name{CommonName: "api.example.com"}, NotBefore: expires.Add(-48 * time.Hour), NotAfter: expires}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	public := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	upload := func(data []byte, status int) dto.AttachmentResponse {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "example.pem")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := writer.WriteField("owner_type", "project"); err != nil {
			t.Fatal(err)
		}
		if err := writer.WriteField("owner_id", project.ID); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", "/attachments", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("upload %d: %s", w.Code, w.Body.String())
		}
		var result struct {
			Data dto.AttachmentResponse `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result.Data
	}
	a := upload(public, 201)
	if len(a.Certificates) != 1 || a.Certificates[0].CommonName != "api.example.com" || a.Certificates[0].ExpiresAt == nil || !a.Certificates[0].ExpiresAt.Equal(expires) {
		t.Fatalf("wrong metadata: %+v", a)
	}
	private, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	upload(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: private}), 400)
	upload(append(append([]byte{}, public...), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private})...), 400)
	upload([]byte("not a certificate"), 400)
	listed := request("GET", "/attachments?owner_type=project&owner_id="+project.ID+"&q=api.example.com&type=x509", "", 200)
	if bytes.Contains(listed, []byte(`"content":`)) || !bytes.Contains(listed, []byte(a.ID)) {
		t.Fatalf("list must contain metadata only: %s", listed)
	}
	request("GET", "/attachments?page=0", "", 400)
	request("GET", "/attachments?owner_type=project", "", 400)
	downloaded := request("GET", "/attachments/"+a.ID+"/download", "", 200)
	if !bytes.Equal(downloaded, public) {
		t.Fatal("original file changed")
	}
	// Concurrent retries must produce exactly one countdown and one unit link.
	var wg sync.WaitGroup
	responses := make(chan *model.Unit, 6)
	failures := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, err := svc.CreateCountdown(a.Certificates[0].ID)
			if err != nil {
				failures <- err
				return
			}
			responses <- u
		}()
	}
	wg.Wait()
	close(responses)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	unitID := ""
	for u := range responses {
		if unitID != "" && unitID != u.ID {
			t.Fatal("duplicate countdown")
		}
		unitID = u.ID
		if u.TargetTime == nil || !u.TargetTime.Equal(expires) || u.ProjectID == nil || *u.ProjectID != project.ID {
			t.Fatalf("wrong countdown: %+v", u)
		}
	}
	unitListPath := "/attachments?owner_type=unit&owner_id=" + unitID
	if !bytes.Contains(request("GET", unitListPath, "", 200), []byte(a.ID)) {
		t.Fatal("countdown missing original certificate")
	}
	owner := `{"owner_type":"unit","owner_id":"` + unitID + `"}`
	request("POST", "/attachments/"+a.ID+"/links", owner, 204)
	request("DELETE", "/attachments/"+a.ID+"/links", owner, 204)
	if bytes.Contains(request("GET", unitListPath, "", 200), []byte(a.ID)) {
		t.Fatal("unlink failed")
	}
	request("POST", "/attachments/"+a.ID+"/links", owner, 204)
	request("POST", "/attachments/"+a.ID+"/links", `{"owner_type":"project","owner_id":"`+uuid.NewString()+`"}`, 404)
	// GPG without expiry is retained and must never produce an invented deadline.
	e, err := openpgp.NewEntity("No expiry", "", "ops@example.com", &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA, Curve: packet.Curve25519})
	if err != nil {
		t.Fatal(err)
	}
	var gpg bytes.Buffer
	if err := e.Serialize(&gpg); err != nil {
		t.Fatal(err)
	}
	b := upload(gpg.Bytes(), 201)
	request("POST", "/certificates/"+b.Certificates[0].ID+"/countdown", "", 400)
	backup := service.NewBackupService(db, "test", "1")
	pkg, err := backup.Export()
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Data.Attachments) != 2 || len(pkg.Data.Certificates) != 2 {
		t.Fatal("backup omitted certificate data")
	}
	raw, err := json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}
	for _, strategy := range []string{dto.BackupStrategyOverwrite, dto.BackupStrategyMerge} {
		if _, err := backup.Import(strategy, raw); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(request("GET", "/attachments/"+a.ID+"/download", "", 200), public) {
			t.Fatal("backup lost original")
		}
		if !bytes.Contains(request("GET", unitListPath, "", 200), []byte(a.ID)) {
			t.Fatal("backup lost links")
		}
	}
	// A backup cannot bypass the public-only upload rule.
	pkg.Data.Attachments[0].Content = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private})
	raw, err = json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := backup.Import(dto.BackupStrategyOverwrite, raw); err == nil {
		t.Fatal("backup imported private material")
	}
	if !bytes.Equal(request("GET", "/attachments/"+a.ID+"/download", "", 200), public) {
		t.Fatal("failed import changed database")
	}
	request("DELETE", "/attachments/"+a.ID, "", 204)
	request("GET", "/attachments/"+a.ID+"/download", "", 404)
	var count int64
	if err := db.Model(&model.Unit{}).Where("id = ?", unitID).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("deleting certificate deleted countdown")
	}
	if err := db.Model(&model.AttachmentLink{}).Where("attachment_id = ?", a.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("orphaned links")
	}
}
