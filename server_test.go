package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GetVocab(t *testing.T) {
	db := memoryDb(t)
	server := NewServer(db)

	dbResult := db.Create(&Vocab{
		Term:           "foo1",
		Translation:    "bar1",
		KnowledgeLevel: 3,
		PracticeAt:     inDays(2),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo2",
		Translation:    "bar2",
		KnowledgeLevel: 1,
		PracticeAt:     inDays(1),
	})
	require.Nil(t, dbResult.Error)

	req, _ := http.NewRequest("GET", "/api/vocab", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	expectedJSON := fmt.Sprintf(`{
		"count": 2,
		"items": [
			{
				"id": 1,
				"term": "foo1",
				"translation": "bar1",
				"knowledgeLevel": 3,
				"practiceAt": "%s"
			},
			{
				"id": 2,
				"term": "foo2",
				"translation": "bar2",
				"knowledgeLevel": 1,
				"practiceAt": "%s"
			}
		]
	}`, inDaysJSON(2), inDaysJSON(1))
	require.JSONEq(t, expectedJSON, rr.Body.String())
}

func Test_GetVocab_Paging(t *testing.T) {
	db := memoryDb(t)
	server := NewServer(db)

	for i := 0; i < 10; i++ {
		dbResult := db.Create(&Vocab{
			Term:           "foo",
			Translation:    "bar",
			KnowledgeLevel: 3,
			PracticeAt:     inDays(2),
		})
		require.Nil(t, dbResult.Error)
	}

	req, _ := http.NewRequest("GET", "/api/vocab?skip=3&take=2", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	expectedJSON := fmt.Sprintf(`{
		"count": 10,
		"items": [
			{
				"id": 4,
				"term": "foo",
				"translation": "bar",
				"knowledgeLevel": 3,
				"practiceAt": "%[1]s"
			},
			{
				"id": 5,
				"term": "foo",
				"translation": "bar",
				"knowledgeLevel": 3,
				"practiceAt": "%[1]s"
			}
		]
	}`, inDaysJSON(2))
	require.JSONEq(t, expectedJSON, rr.Body.String())
}

func Test_GetVocab_Search(t *testing.T) {
	db := memoryDb(t)
	server := NewServer(db)

	dbResult := db.Create(&Vocab{
		Term:           "guten tag",
		Translation:    "good day",
		KnowledgeLevel: 1,
		PracticeAt:     inDays(1),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "apfel kuchen",
		Translation:    "apple cake",
		KnowledgeLevel: 1,
		PracticeAt:     inDays(1),
	})
	require.Nil(t, dbResult.Error)

	req, _ := http.NewRequest("GET", "/api/vocab?term=guten", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	expectedJSON := fmt.Sprintf(`{
		"count": 1,
		"items": [
			{
				"id": 1,
				"term": "guten tag",
				"translation": "good day",
				"knowledgeLevel": 1,
				"practiceAt": "%s"
			}
		]
	}`, inDaysJSON(1))
	require.JSONEq(t, expectedJSON, rr.Body.String())

	req, _ = http.NewRequest("GET", "/api/vocab?translation=apple%20cake", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	expectedJSON = fmt.Sprintf(`{
		"count": 1,
		"items": [
			{
				"id": 2,
				"term": "apfel kuchen",
				"translation": "apple cake",
				"knowledgeLevel": 1,
				"practiceAt": "%s"
			}
		]
	}`, inDaysJSON(1))
	require.JSONEq(t, expectedJSON, rr.Body.String())

	req, _ = http.NewRequest("GET", "/api/vocab?term=guten&translation=apple%20cake", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{
		"count": 0,
		"items": []
	}`, rr.Body.String())

	req, _ = http.NewRequest("GET", "/api/vocab?term=guten&translation=apple%20cake&mode=or", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	expectedJSON = fmt.Sprintf(`{
		"count": 2,
		"items": [
			{
				"id": 2,
				"term": "apfel kuchen",
				"translation": "apple cake",
				"knowledgeLevel": 1,
				"practiceAt": "%[1]s"
			},
			{
				"id": 1,
				"term": "guten tag",
				"translation": "good day",
				"knowledgeLevel": 1,
				"practiceAt": "%[1]s"
			}
		]
	}`, inDaysJSON(1))
	require.JSONEq(t, expectedJSON, rr.Body.String())
}

func Test_GetVocab_OrderBy(t *testing.T) {
	db := memoryDb(t)
	server := NewServer(db)

	dbResult := db.Create(&Vocab{
		Term:           "foo1",
		Translation:    "bar1",
		KnowledgeLevel: 3,
		PracticeAt:     inDays(2),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo2",
		Translation:    "bar2",
		KnowledgeLevel: 5,
		PracticeAt:     inDays(4),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo3",
		Translation:    "bar3",
		KnowledgeLevel: 1,
		PracticeAt:     inDays(3),
	})
	require.Nil(t, dbResult.Error)

	req, _ := http.NewRequest("GET", "/api/vocab?order_by=knowledge_level", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	expectedJSON := fmt.Sprintf(`{
		"count": 3,
		"items": [
			{
				"id": 3,
				"term": "foo3",
				"translation": "bar3",
				"knowledgeLevel": 1,
				"practiceAt": "%[3]s"
			},
			{
				"id": 1,
				"term": "foo1",
				"translation": "bar1",
				"knowledgeLevel": 3,
				"practiceAt": "%[1]s"
			},
			{
				"id": 2,
				"term": "foo2",
				"translation": "bar2",
				"knowledgeLevel": 5,
				"practiceAt": "%[2]s"
			}
		]
	}`, inDaysJSON(2), inDaysJSON(4), inDaysJSON(3))
	require.JSONEq(t, expectedJSON, rr.Body.String())

	req, _ = http.NewRequest("GET", "/api/vocab?order_by=knowledge_level_desc", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	expectedJSON = fmt.Sprintf(`{
		"count": 3,
		"items": [
			{
				"id": 2,
				"term": "foo2",
				"translation": "bar2",
				"knowledgeLevel": 5,
				"practiceAt": "%[2]s"
			},
			{
				"id": 1,
				"term": "foo1",
				"translation": "bar1",
				"knowledgeLevel": 3,
				"practiceAt": "%[1]s"
			},
			{
				"id": 3,
				"term": "foo3",
				"translation": "bar3",
				"knowledgeLevel": 1,
				"practiceAt": "%[3]s"
			}
		]
	}`, inDaysJSON(2), inDaysJSON(4), inDaysJSON(3))
	require.JSONEq(t, expectedJSON, rr.Body.String())

	req, _ = http.NewRequest("GET", "/api/vocab?order_by=practice_at", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	expectedJSON = fmt.Sprintf(`{
		"count": 3,
		"items": [
			{
				"id": 1,
				"term": "foo1",
				"translation": "bar1",
				"knowledgeLevel": 3,
				"practiceAt": "%[1]s"
			},
			{
				"id": 3,
				"term": "foo3",
				"translation": "bar3",
				"knowledgeLevel": 1,
				"practiceAt": "%[3]s"
			},
			{
				"id": 2,
				"term": "foo2",
				"translation": "bar2",
				"knowledgeLevel": 5,
				"practiceAt": "%[2]s"
			}
		]
	}`, inDaysJSON(2), inDaysJSON(4), inDaysJSON(3))
	require.JSONEq(t, expectedJSON, rr.Body.String())
}

func Test_PostVocab(t *testing.T) {
	db := memoryDb(t)
	server := NewServer(db)

	var body bytes.Buffer
	_, err := body.WriteString(`{
		"term": "foo",
		"translation": "bar"
	}`)
	require.Nil(t, err)

	req, _ := http.NewRequest("POST", "/api/vocab", &body)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{"id": 1}`, rr.Body.String())

	v := Vocab{}
	dbResult := db.First(&v)
	require.Nil(t, dbResult.Error)
	require.Equal(t, "foo", v.Term)
	require.Equal(t, "bar", v.Translation)
	require.Equal(t, uint(0), v.KnowledgeLevel)
	require.True(t, v.PracticeAt.Equal(inDays(0)))
}

func Test_DeleteVocab(t *testing.T) {
	db := memoryDb(t)
	server := NewServer(db)

	dbResult := db.Create(&Vocab{
		Term:           "foo",
		Translation:    "bar",
		KnowledgeLevel: 3,
		PracticeAt:     inDays(2),
	})
	require.Nil(t, dbResult.Error)

	req, _ := http.NewRequest("DELETE", "/api/vocab/1", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var rowCount int64
	dbResult = db.Model(&Vocab{}).Count(&rowCount)
	require.Nil(t, dbResult.Error)
	require.Equal(t, int64(0), rowCount)
}

func Test_GetPractice(t *testing.T) {
	db := memoryDb(t)
	server := NewServer(db)

	dbResult := db.Create(&Vocab{
		Term:           "foo1",
		Translation:    "bar1",
		KnowledgeLevel: 2,
		PracticeAt:     inDays(0),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo2",
		Translation:    "bar2",
		KnowledgeLevel: 1,
		PracticeAt:     inDays(1),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo3",
		Translation:    "bar3",
		KnowledgeLevel: 4,
		PracticeAt:     inDays(-1),
	})
	require.Nil(t, dbResult.Error)

	req, _ := http.NewRequest("GET", "/api/practice", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	expectedJSON := fmt.Sprintf(`[
		{
			"id": 3,
			"term": "foo3",
			"translation": "bar3",
			"knowledgeLevel": 4,
			"practiceAt": "%s"
		},
		{
			"id": 1,
			"term": "foo1",
			"translation": "bar1",
			"knowledgeLevel": 2,
			"practiceAt": "%s"
		}
	]`, inDaysJSON(-1), inDaysJSON(0))
	require.JSONEq(t, expectedJSON, rr.Body.String())
}

func Test_GetCountPractice(t *testing.T) {
	db := memoryDb(t)
	server := NewServer(db)

	dbResult := db.Create(&Vocab{
		Term:           "foo1",
		Translation:    "bar1",
		KnowledgeLevel: 2,
		PracticeAt:     inDays(0),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo2",
		Translation:    "bar2",
		KnowledgeLevel: 1,
		PracticeAt:     inDays(1),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo3",
		Translation:    "bar3",
		KnowledgeLevel: 4,
		PracticeAt:     inDays(-1),
	})
	require.Nil(t, dbResult.Error)

	req, _ := http.NewRequest("GET", "/api/practice/count", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.JSONEq(t, `{"count": 2}`, rr.Body.String())
}

func Test_PostPractice(t *testing.T) {
	db := memoryDb(t)
	server := NewServer(db)

	dbResult := db.Create(&Vocab{
		Term:           "foo1",
		Translation:    "bar1",
		KnowledgeLevel: 0,
		PracticeAt:     inDays(0),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo2",
		Translation:    "bar2",
		KnowledgeLevel: 5,
		PracticeAt:     inDays(0),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo3",
		Translation:    "bar3",
		KnowledgeLevel: 2,
		PracticeAt:     inDays(0),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo4",
		Translation:    "bar4",
		KnowledgeLevel: 7,
		PracticeAt:     inDays(0),
	})
	require.Nil(t, dbResult.Error)

	var body bytes.Buffer
	_, err := body.WriteString(`[
		{
			"id": 1,
			"passed": false
		},
		{
			"id": 2,
			"passed": false
		},
		{
			"id": 3,
			"passed": true
		},
		{
			"id": 4,
			"passed": true
		}
	]`)
	require.Nil(t, err)

	req, _ := http.NewRequest("POST", "/api/practice", &body)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var v Vocab
	dbResult = db.First(&v, 1)
	require.Nil(t, dbResult.Error)
	require.Equal(t, uint(0), v.KnowledgeLevel)
	require.True(t, v.PracticeAt.Equal(inDays(1)))

	v = Vocab{}
	dbResult = db.First(&v, 2)
	require.Nil(t, dbResult.Error)
	require.Equal(t, uint(4), v.KnowledgeLevel)
	require.True(t, v.PracticeAt.Equal(inDays(1)))

	v = Vocab{}
	dbResult = db.First(&v, 3)
	require.Nil(t, dbResult.Error)
	require.Equal(t, uint(3), v.KnowledgeLevel)
	require.True(t, v.PracticeAt.Equal(inDays(4)))

	v = Vocab{}
	dbResult = db.First(&v, 4)
	require.Nil(t, dbResult.Error)
	require.Equal(t, uint(7), v.KnowledgeLevel)
	require.True(t, v.PracticeAt.Equal(inDays(64)))
}
