package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/repository"
	"paradigm-reboot-prober-go/internal/service"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.Song{},
		&model.SongLevel{},
		&model.PlayRecord{},
		&model.BestPlayRecord{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func setupTestRouter() (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	db := setupTestDB(&testing.T{})
	return r, db
}

func performRequest(r http.Handler, method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type testEnv struct {
	db            *gorm.DB
	userService   *service.UserService
	songService   *service.SongService
	recordService *service.RecordService
	userCtrl      *UserController
	songCtrl      *SongController
	recordCtrl    *RecordController
	uploadCtrl    *UploadController
}

func setupEnv(t *testing.T) *testEnv {
	// Initialize config for tests
	config.GlobalConfig.Upload.CSVPath = "./uploads/csv"
	config.GlobalConfig.Upload.ImgPath = "./uploads/img"
	config.GlobalConfig.Auth.SecretKey = "testsecret"

	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	songRepo := repository.NewSongRepository(db)
	recordRepo := repository.NewRecordRepository(db)

	userService := service.NewUserService(userRepo)
	songService := service.NewSongService(songRepo)
	recordService := service.NewRecordService(recordRepo, songRepo)

	return &testEnv{
		db:            db,
		userService:   userService,
		songService:   songService,
		recordService: recordService,
		userCtrl:      NewUserController(userService),
		songCtrl:      NewSongController(songService),
		recordCtrl:    NewRecordController(recordService, userService),
		uploadCtrl:    NewUploadController(userService),
	}
}
