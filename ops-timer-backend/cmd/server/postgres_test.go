package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/repository"
	"ops-timer-backend/internal/service"
)

// Use an isolated schema so integration tests never clear existing application tables.
func testPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TASK_MANAGER_TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("set TASK_MANAGER_TEST_DATABASE_DSN to run PostgreSQL integration tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Error(err)
		}
		sqlDB.Close()
	})
	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	autoMigrate(db)
	autoMigrate(db) // Startup must also work with existing PostgreSQL tables.
	return db
}

func TestPostgresTodo(t *testing.T) {
	db := testPostgres(t)
	svc := service.NewTodoService(repository.NewTodoRepository(db), repository.NewTodoGroupRepository(db))
	h := handler.NewTodoHandler(svc)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/todos", h.Create)
	r.PUT("/todos/:id", h.Update)
	r.GET("/todos/:id", h.Get)
	r.GET("/todos", h.List)
	r.DELETE("/todos/:id", h.Delete)
	request := func(method, path, body string, status int) json.RawMessage {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s: status %d, want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		if status == 204 {
			return nil
		}
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope.Data
	}
	decode := func(raw json.RawMessage) dto.TodoResponse {
		t.Helper()
		var todo dto.TodoResponse
		if err := json.Unmarshal(raw, &todo); err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"importance", "difficulty"} {
			if _, ok := fields[key]; !ok {
				t.Fatalf("missing %s", key)
			}
		}
		return todo
	}
	todo := decode(request("POST", "/todos", `{"title":"default"}`, 201))
	if todo.Importance != 0.5 || todo.Difficulty != 0.5 {
		t.Fatal("incorrect defaults")
	}
	path := "/todos/" + todo.ID
	todo = decode(request("PUT", path, `{"importance":0.375,"difficulty":1}`, 200))
	if todo.Importance != 0.375 || todo.Difficulty != 1 {
		t.Fatal("scores not updated")
	}
	todo = decode(request("PUT", path, `{"title":"renamed"}`, 200))
	if todo.Importance != 0.375 || todo.Difficulty != 1 {
		t.Fatal("omitted scores changed")
	}
	todo = decode(request("GET", path, "", 200))
	if todo.Importance != 0.375 || todo.Difficulty != 1 {
		t.Fatal("scores not persisted")
	}
	for _, field := range []string{"importance", "difficulty"} {
		for _, value := range []string{"-0.01", "1.01", `"0.5"`, "true", "1e999"} {
			body := fmt.Sprintf(`{"title":"invalid","%s":%s}`, field, value)
			request("POST", "/todos", body, 400)
			request("PUT", path, body, 400)
		}
		for _, value := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1), math.Inf(-1)} {
			if err := db.Model(&model.Todo{}).Where("id = ?", todo.ID).Update(field, value).Error; err == nil {
				t.Fatalf("database accepted %s=%v", field, value)
			}
		}
		if err := db.Model(&model.Todo{}).Where("id = ?", todo.ID).Update(field, nil).Error; err == nil {
			t.Fatalf("database accepted NULL %s", field)
		}
	}
	var list []dto.TodoResponse
	if err := json.Unmarshal(request("GET", "/todos", "", 200), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Importance != 0.375 || list[0].Difficulty != 1 {
		t.Fatalf("unexpected list: %+v", list)
	}
	backup := service.NewBackupService(db, "test", "1")
	pkg, err := backup.Export()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}
	for _, strategy := range []string{dto.BackupStrategyOverwrite, dto.BackupStrategyMerge} {
		if _, err := backup.Import(strategy, raw); err != nil {
			t.Fatal(err)
		}
		restored := decode(request("GET", path, "", 200))
		if restored.Importance != 0.375 || restored.Difficulty != 1 {
			t.Fatal("backup lost scores")
		}
	}
	todo = decode(request("PUT", path, `{"importance":0,"difficulty":0}`, 200))
	todo = decode(request("GET", path, "", 200))
	if todo.Importance != 0 || todo.Difficulty != 0 {
		t.Fatal("cannot reset scores to zero")
	}
	boundary := decode(request("POST", "/todos", `{"title":"boundary","importance":1,"difficulty":0.625}`, 201))
	if boundary.Importance != 1 || boundary.Difficulty != 0.625 {
		t.Fatal("create lost scores")
	}
	request("DELETE", path, "", 204)
	request("GET", path, "", 404)
}

func TestPostgresAuthPersistence(t *testing.T) {
	db := testPostgres(t)
	attempts := repository.NewLoginAttemptRepository(db)
	for i := 0; i < 3; i++ {
		if err := attempts.Increment("admin", 3); err != nil {
			t.Fatal(err)
		}
	}
	attempt, err := attempts.Get("admin")
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Count != 3 || attempt.LockedAt.IsZero() {
		t.Fatalf("unexpected login attempt: %+v", attempt)
	}
	if err := attempts.Reset("admin"); err != nil {
		t.Fatal(err)
	}
	tokens := repository.NewRevokedTokenRepository(db)
	if err := tokens.Add("token", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if !tokens.Exists("token") {
		t.Fatal("revoked token was not persisted")
	}
	if err := tokens.Add("expired", time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := tokens.Cleanup(); err != nil {
		t.Fatal(err)
	}
	loaded, err := tokens.LoadAll()
	if err != nil || len(loaded) != 1 {
		t.Fatalf("unexpected blacklist: %v, %v", loaded, err)
	}
}
