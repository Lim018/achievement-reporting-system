package test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"go-fiber/app/model"
	"go-fiber/app/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestCreateReference(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	repo := repository.NewAchievementRefRepo(db)

	t.Run("Create Reference Success", func(t *testing.T) {
		studentID := "student-uuid-1"
		mongoHex := "507f1f77bcf86cd799439011"
		expectedRefID := "ref-uuid-1"

		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO achievement_references`)).
			WithArgs(studentID, mongoHex).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedRefID))

		refID, err := repo.CreateReference(studentID, mongoHex)

		assert.NoError(t, err)
		assert.Equal(t, expectedRefID, refID)
	})

	t.Run("Create Reference Database Error", func(t *testing.T) {
		studentID := "student-uuid-1"
		mongoHex := "507f1f77bcf86cd799439011"

		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO achievement_references`)).
			WithArgs(studentID, mongoHex).
			WillReturnError(sql.ErrConnDone)

		refID, err := repo.CreateReference(studentID, mongoHex)

		assert.Error(t, err)
		assert.Empty(t, refID)
		assert.Equal(t, sql.ErrConnDone, err)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetReference(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	repo := repository.NewAchievementRefRepo(db)
	fixedTime := time.Now()

	cols := []string{
		"id", "student_id", "mongo_achievement_id", "status",
		"submitted_at", "verified_at", "verified_by", "rejection_note",
		"created_at", "updated_at",
	}

	t.Run("Get Reference Success", func(t *testing.T) {
		refID := "ref-uuid-1"
		studentID := "student-uuid-1"
		mongoID := "507f1f77bcf86cd799439011"

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, student_id, mongo_achievement_id`)).
			WithArgs(refID).
			WillReturnRows(sqlmock.NewRows(cols).AddRow(
				refID, studentID, mongoID, "draft",
				nil, nil, nil, nil,
				fixedTime, fixedTime,
			))

		result, err := repo.GetReference(refID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, refID, result.ReferenceID)
		assert.Equal(t, studentID, result.StudentID)
		assert.Equal(t, mongoID, result.MongoID)
		assert.Equal(t, "draft", result.ReferenceStatus)
	})

	t.Run("Get Reference Not Found", func(t *testing.T) {
		refID := "non-existent"

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, student_id, mongo_achievement_id`)).
			WithArgs(refID).
			WillReturnError(sql.ErrNoRows)

		result, err := repo.GetReference(refID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, sql.ErrNoRows, err)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSubmitReference(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	repo := repository.NewAchievementRefRepo(db)

	t.Run("Submit Reference Success", func(t *testing.T) {
		refID := "ref-uuid-1"

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE achievement_references`)).
			WithArgs(refID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.SubmitReference(refID)

		assert.NoError(t, err)
	})

	t.Run("Submit Reference Not Found", func(t *testing.T) {
		refID := "non-existent"

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE achievement_references`)).
			WithArgs(refID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.SubmitReference(refID)

		assert.NoError(t, err)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestVerifyReference(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	repo := repository.NewAchievementRefRepo(db)

	t.Run("Verify Reference Success", func(t *testing.T) {
		refID := "ref-uuid-1"
		verifierID := "advisor-uuid-1"

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE achievement_references`)).
			WithArgs(verifierID, refID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.VerifyReference(refID, verifierID)

		assert.NoError(t, err)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRejectReference(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	repo := repository.NewAchievementRefRepo(db)

	t.Run("Reject Reference Success", func(t *testing.T) {
		refID := "ref-uuid-1"
		verifierID := "advisor-uuid-1"
		note := "Data tidak lengkap"

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE achievement_references`)).
			WithArgs(verifierID, note, refID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.RejectReference(refID, verifierID, note)

		assert.NoError(t, err)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestListForStudent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	repo := repository.NewAchievementRefRepo(db)
	fixedTime := time.Now()

	cols := []string{
		"id", "mongo_achievement_id", "status",
		"submitted_at", "verified_at", "verified_by", "rejection_note",
		"created_at", "updated_at",
	}

	t.Run("List For Student Success", func(t *testing.T) {
		studentID := "student-uuid-1"

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mongo_achievement_id, status`)).
			WithArgs(studentID).
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow("ref-1", "mongo-1", "draft", nil, nil, nil, nil, fixedTime, fixedTime).
				AddRow("ref-2", "mongo-2", "submitted", fixedTime, nil, nil, nil, fixedTime, fixedTime).
				AddRow("ref-3", "mongo-3", "verified", fixedTime, fixedTime, "advisor-1", nil, fixedTime, fixedTime),
			)

		result, err := repo.ListForStudent(studentID)

		assert.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, "ref-1", result[0].ReferenceID)
		assert.Equal(t, "draft", result[0].ReferenceStatus)
		assert.Equal(t, "verified", result[2].ReferenceStatus)
	})

	t.Run("List For Student Empty", func(t *testing.T) {
		studentID := "student-uuid-no-achievements"

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mongo_achievement_id, status`)).
			WithArgs(studentID).
			WillReturnRows(sqlmock.NewRows(cols))

		result, err := repo.ListForStudent(studentID)

		assert.NoError(t, err)
		assert.Len(t, result, 0)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestListForAdvisor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	repo := repository.NewAchievementRefRepo(db)
	fixedTime := time.Now()

	cols := []string{
		"id", "mongo_achievement_id", "status",
		"submitted_at", "verified_at", "verified_by", "rejection_note",
		"created_at", "updated_at",
	}

	t.Run("List For Advisor Success", func(t *testing.T) {
		advisorUserID := "advisor-user-uuid-1"

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT ar.id, ar.mongo_achievement_id`)).
			WithArgs(advisorUserID).
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow("ref-1", "mongo-1", "submitted", fixedTime, nil, nil, nil, fixedTime, fixedTime).
				AddRow("ref-2", "mongo-2", "verified", fixedTime, fixedTime, advisorUserID, nil, fixedTime, fixedTime),
			)

		result, err := repo.ListForAdvisor(advisorUserID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "submitted", result[0].ReferenceStatus)
		assert.Equal(t, "verified", result[1].ReferenceStatus)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSoftDeleteReference(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %v", err)
	}
	defer db.Close()

	repo := repository.NewAchievementRefRepo(db)

	t.Run("Soft Delete Success", func(t *testing.T) {
		refID := "ref-uuid-1"

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE achievement_references`)).
			WithArgs(refID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.SoftDeleteReference(refID)

		assert.NoError(t, err)
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestMongoAchievementOperations(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("Create Achievement Success", func(mt *mtest.T) {
		repo := repository.NewAchievementMongoRepo(mt.DB)

		ach := model.Achievement{
			StudentID:       "student-uuid-1",
			AchievementType: "competition",
			Title:           "Test Competition",
			Description:     "Test Description",
			Points:          100,
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		mongoID, err := repo.Create(context.Background(), ach)

		assert.NoError(mt, err)
		assert.NotEmpty(mt, mongoID)
	})

	mt.Run("FindByHexID Success", func(mt *mtest.T) {
		repo := repository.NewAchievementMongoRepo(mt.DB)

		objectID := primitive.NewObjectID()
		hexID := objectID.Hex()

		expectedDoc := mtest.CreateCursorResponse(
			1,
			"test.achievement_records",
			mtest.FirstBatch,
			primitive.D{
				{Key: "_id", Value: objectID},
				{Key: "studentId", Value: "student-uuid-1"},
				{Key: "achievementType", Value: "competition"},
				{Key: "title", Value: "Test Achievement"},
				{Key: "points", Value: 100},
			},
		)

		mt.AddMockResponses(expectedDoc)

		result, err := repo.FindByHexID(context.Background(), hexID)

		assert.NoError(mt, err)
		assert.NotNil(mt, result)
		assert.Equal(mt, "Test Achievement", result.Title)
		assert.Equal(mt, 100, result.Points)
	})

	mt.Run("UpdateByHexID Success", func(mt *mtest.T) {
		repo := repository.NewAchievementMongoRepo(mt.DB)

		hexID := primitive.NewObjectID().Hex()
		update := map[string]interface{}{
			"title":  "Updated Title",
			"points": 150,
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err := repo.UpdateByHexID(context.Background(), hexID, update)

		assert.NoError(mt, err)
	})

	mt.Run("DeleteByHexID Success", func(mt *mtest.T) {
		repo := repository.NewAchievementMongoRepo(mt.DB)

		hexID := primitive.NewObjectID().Hex()

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err := repo.DeleteByHexID(context.Background(), hexID)

		assert.NoError(mt, err)
	})
}

func TestAchievementCache(t *testing.T) {
	cache := NewAchievementCacheForTest(5 * time.Minute)

	t.Run("Set and Get Cache", func(t *testing.T) {
		key := "test-key"
		ach := &model.Achievement{
			ID:              primitive.NewObjectID(),
			StudentID:       "student-1",
			AchievementType: "competition",
			Title:           "Cached Achievement",
			Points:          100,
		}

		cache.Set(key, ach)
		retrieved, found := cache.Get(key)

		assert.True(t, found)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "Cached Achievement", retrieved.Title)
		assert.Equal(t, 100, retrieved.Points)
	})

	t.Run("Get Non-Existent Key", func(t *testing.T) {
		retrieved, found := cache.Get("non-existent")

		assert.False(t, found)
		assert.Nil(t, retrieved)
	})

	t.Run("Delete Cache", func(t *testing.T) {
		key := "delete-test"
		ach := &model.Achievement{
			Title: "To Be Deleted",
		}

		cache.Set(key, ach)
		cache.Delete(key)
		retrieved, found := cache.Get(key)

		assert.False(t, found)
		assert.Nil(t, retrieved)
	})

	t.Run("Cache Expiration", func(t *testing.T) {
		shortCache := NewAchievementCacheForTest(100 * time.Millisecond)
		key := "expire-test"
		ach := &model.Achievement{Title: "Will Expire"}

		shortCache.Set(key, ach)

		retrieved, found := shortCache.Get(key)
		assert.True(t, found)
		assert.NotNil(t, retrieved)

		time.Sleep(150 * time.Millisecond)

		retrieved, found = shortCache.Get(key)
		assert.False(t, found)
		assert.Nil(t, retrieved)
	})
}

func NewAchievementCacheForTest(ttl time.Duration) *AchievementCacheTest {
	return &AchievementCacheTest{
		data: make(map[string]*model.Achievement),
		exp:  make(map[string]time.Time),
		ttl:  ttl,
	}
}

type AchievementCacheTest struct {
	data map[string]*model.Achievement
	exp  map[string]time.Time
	ttl  time.Duration
}

func (c *AchievementCacheTest) Get(key string) (*model.Achievement, bool) {
	if exp, exists := c.exp[key]; exists {
		if time.Now().After(exp) {
			return nil, false
		}
		if val, ok := c.data[key]; ok {
			return val, true
		}
	}
	return nil, false
}

func (c *AchievementCacheTest) Set(key string, val *model.Achievement) {
	c.data[key] = val
	c.exp[key] = time.Now().Add(c.ttl)
}

func (c *AchievementCacheTest) Delete(key string) {
	delete(c.data, key)
	delete(c.exp, key)
}

func BenchmarkListForStudent(b *testing.B) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewAchievementRefRepo(db)
	fixedTime := time.Now()

	cols := []string{
		"id", "mongo_achievement_id", "status",
		"submitted_at", "verified_at", "verified_by", "rejection_note",
		"created_at", "updated_at",
	}

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mongo_achievement_id, status`)).
			WithArgs("student-uuid-1").
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow("ref-1", "mongo-1", "draft", nil, nil, nil, nil, fixedTime, fixedTime).
				AddRow("ref-2", "mongo-2", "verified", fixedTime, fixedTime, "advisor-1", nil, fixedTime, fixedTime),
			)

		repo.ListForStudent("student-uuid-1")
	}
}

func BenchmarkCacheOperations(b *testing.B) {
	cache := NewAchievementCacheForTest(5 * time.Minute)
	ach := &model.Achievement{
		Title:  "Benchmark Test",
		Points: 100,
	}

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Set("key", ach)
		}
	})

	b.Run("Get", func(b *testing.B) {
		cache.Set("key", ach)
		for i := 0; i < b.N; i++ {
			cache.Get("key")
		}
	})
}