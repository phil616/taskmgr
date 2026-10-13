package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/repository"
	"ops-timer-backend/internal/service"
	"strings"
	"testing"
)

func TestNoteDraftRecoveryAndAtomicPublish(t *testing.T) {
	db := testPostgres(t)
	svc := service.NewNoteService(repository.NewNoteRepository(db), repository.NewNoteGroupRepository(db))
	h := handler.NewNoteHandler(svc)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/note-drafts", h.SaveDraft)
	r.GET("/note-drafts", h.ListDrafts)
	r.PUT("/note-drafts/:id", h.SaveDraft)
	r.POST("/note-drafts/:id/publish", h.PublishDraft)
	r.DELETE("/note-drafts/:id", h.DeleteDraft)
	call := func(method, path, body string, status int, target any) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		if target != nil {
			var envelope struct {
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(envelope.Data, target); err != nil {
				t.Fatal(err)
			}
		}
	}
	original, err := svc.Create(&dto.CreateNoteRequest{Title: "Original", Content: "unchanged"})
	if err != nil {
		t.Fatal(err)
	}
	var draft model.NoteDraft
	call("POST", "/note-drafts", `{}`, 201, &draft)
	call("POST", "/note-drafts/"+draft.ID+"/publish", `{}`, 400, nil)
	updated, _ := json.Marshal(map[string]any{"note_id": original.ID, "title": "Draft", "content": "# Heading\n\n```go\nfunc main() {}\n```", "tags": []string{"恢复"}})
	call("PUT", "/note-drafts/"+draft.ID, string(updated), 200, &draft)
	unchanged, _ := svc.GetByID(original.ID)
	if unchanged.Content != "unchanged" {
		t.Fatal("saving draft changed original note")
	}
	var list []model.NoteDraft
	call("GET", "/note-drafts", "", 200, &list)
	if len(list) != 1 || list[0].Content != draft.Content {
		t.Fatal("draft snapshot was not preserved")
	}
	backup, err := service.NewBackupService(db, "test", "1.0.0").Export()
	if err != nil {
		t.Fatal(err)
	}
	if len(backup.Data.NoteDrafts) != 1 {
		t.Fatal("backup omitted draft")
	}
	encoded, err := json.Marshal(backup)
	if err != nil {
		t.Fatal(err)
	}
	for _, strategy := range []string{dto.BackupStrategyOverwrite, dto.BackupStrategyMerge} {
		imported, err := service.NewBackupService(db, "test", "1.0.0").Import(strategy, encoded)
		if err != nil || imported.Stats.NoteDrafts != 1 {
			t.Fatalf("draft backup import (%s): %v", strategy, err)
		}
		call("GET", "/note-drafts", "", 200, &list)
		if len(list) != 1 || list[0].ID != draft.ID || list[0].Content != draft.Content {
			t.Fatalf("draft backup recovery (%s) lost data", strategy)
		}
	}
	bad := `{"note_id":"` + original.ID + `","title":"Publish","content":"edited","group_id":"00000000-0000-4000-8000-000000000001"}`
	call("POST", "/note-drafts/"+draft.ID+"/publish", bad, 404, nil)
	if _, err := svc.ListDrafts(); err != nil {
		t.Fatal(err)
	}
	unchanged, _ = svc.GetByID(original.ID)
	if unchanged.Content != "unchanged" {
		t.Fatal("failed publish modified original")
	}
	var result dto.NoteResponse
	body := `{"note_id":"` + original.ID + `","title":"Publish","content":"edited","tags":[]}`
	call("POST", "/note-drafts/"+draft.ID+"/publish", body, 200, &result)
	if result.ID != original.ID || result.Content != "edited" {
		t.Fatal("draft did not update original note")
	}
	call("GET", "/note-drafts", "", 200, &list)
	if len(list) != 0 {
		t.Fatal("published draft still exists")
	}
	call("POST", "/note-drafts/"+draft.ID+"/publish", body, 404, nil)
	call("PUT", "/note-drafts/"+draft.ID, `{}`, 404, nil)
	call("POST", "/note-drafts", `{"title":"New","content":"new body"}`, 201, &draft)
	call("POST", "/note-drafts/"+draft.ID+"/publish", `{"title":"New","content":"new body"}`, 200, &result)
	if result.ID == original.ID {
		t.Fatal("new draft overwrote existing note")
	}
	call("POST", "/note-drafts", `{}`, 201, &draft)
	call("DELETE", "/note-drafts/"+draft.ID, "", 204, nil)
	call("DELETE", "/note-drafts/"+draft.ID, "", 404, nil)
}
