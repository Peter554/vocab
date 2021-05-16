package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "embed"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Server struct {
	router *mux.Router
}

func NewServer(db *gorm.DB) *Server {
	router := mux.NewRouter()
	router.Use(errorHandlingMiddleware)

	api := router.PathPrefix("/api").Subrouter()
	vocabHandler := &vocabHandler{db: db}
	api.HandleFunc("/vocab", vocabHandler.get).Methods("GET")
	api.HandleFunc("/vocab", vocabHandler.post).Methods("POST")
	api.HandleFunc("/vocab/{id:\\d+}", vocabHandler.delete).Methods("DELETE")
	practiceHandler := &practiceHandler{db: db}
	api.HandleFunc("/practice", practiceHandler.get).Methods("GET")
	api.HandleFunc("/practice/count", practiceHandler.getCount).Methods("GET")
	api.HandleFunc("/practice", practiceHandler.post).Methods("POST")
	router.PathPrefix("/").Handler(http.HandlerFunc(serveSPA))

	return &Server{
		router: router,
	}
}

func (a *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

//go:embed www
var www embed.FS

func serveSPA(w http.ResponseWriter, r *http.Request) {
	_, err := os.Stat("www")
	if err == nil {
		// dev
		_, err = os.Stat(filepath.Join("www", r.URL.Path))
		if err == nil {
			http.FileServer(http.Dir("www")).ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, "www/index.html")
	} else {
		// prod
		_, err = www.Open(filepath.Join("www", r.URL.Path))
		if err == nil {
			files, err := fs.Sub(www, "www")
			check(err)
			http.FileServer(http.FS(files)).ServeHTTP(w, r)
			return
		}
		b, err := www.ReadFile("index.html")
		check(err)
		_, err = w.Write(b)
		check(err)
	}
}

func errorHandlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
				log.Println(err)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type vocabHandler struct {
	db *gorm.DB
}

func (h *vocabHandler) get(w http.ResponseWriter, r *http.Request) {
	qp := &QueryParams{r}

	q := h.db.Model(&Vocab{})

	termQp := qp.Str("term", "")
	translationQp := qp.Str("translation", "")
	if termQp != "" && translationQp != "" {
		if qp.Str("mode", "") == "or" {
			q = q.Where("term like ? or translation like ?", like(termQp), like(translationQp))
		} else {
			q = q.Where("term like ? and translation like ?", like(termQp), like(translationQp))
		}
	} else if termQp != "" {
		q = q.Where("term like ?", like(termQp))
	} else if translationQp != "" {
		q = q.Where("translation like ?", like(translationQp))
	}

	var count int64
	dbResult := q.Count(&count)
	check(dbResult.Error)

	orderBy, orderByQp := "term", qp.Str("order_by", "")
	if orderByQp == "knowledge_level" {
		orderBy = "knowledge_level"
	} else if orderByQp == "knowledge_level_desc" {
		orderBy = "knowledge_level desc"
	} else if orderByQp == "practice_at" {
		orderBy = "practice_at"
	} else if orderByQp == "practice_at_desc" {
		orderBy = "practice_at desc"
	}

	vocabs := make([]Vocab, 0)
	dbResult = q.
		Order(orderBy + ", term").
		Offset(qp.Int("skip", 0)).
		Limit(min(qp.Int("take", 10), 50)).
		Find(&vocabs)
	check(dbResult.Error)

	err := writeJSON(w, map[string]interface{}{
		"count": count,
		"items": vocabs,
	})
	check(err)
}

func (h *vocabHandler) post(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	check(err)

	var requestData map[string]string
	err = json.Unmarshal(body, &requestData)
	check(err)

	if term, ok := requestData["term"]; !ok || term == "" {
		http.Error(w, "term is required", http.StatusBadRequest)
		return
	}

	if translation, ok := requestData["translation"]; !ok || translation == "" {
		http.Error(w, "translation is required", http.StatusBadRequest)
		return
	}

	vocab := &Vocab{
		Term:           requestData["term"],
		Translation:    requestData["translation"],
		KnowledgeLevel: 0,
		PracticeAt:     inDays(0),
	}
	dbResult := h.db.Create(vocab)
	check(dbResult.Error)

	err = writeJSON(w, map[string]uint{"id": vocab.ID})
	check(err)
}

func (h *vocabHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	dbResult := h.db.Delete(&Vocab{}, id)
	check(dbResult.Error)
}

type practiceHandler struct {
	db *gorm.DB
}

func (h *practiceHandler) get(w http.ResponseWriter, r *http.Request) {
	vocabs := make([]Vocab, 0)
	dbResult := h.db.
		Model(&Vocab{}).
		Where("practice_at < ?", time.Now()).
		Order("practice_at").
		Limit(10).
		Find(&vocabs)
	check(dbResult.Error)

	err := writeJSON(w, vocabs)
	check(err)
}

func (h *practiceHandler) getCount(w http.ResponseWriter, r *http.Request) {
	var count int64
	dbResult := h.db.
		Model(&Vocab{}).
		Where("practice_at < ?", time.Now()).
		Count(&count)
	check(dbResult.Error)

	err := writeJSON(w, struct {
		Count int64 `json:"count"`
	}{count})
	check(err)
}

var maxKnowledge uint = 7
var knowledgeToPracticeMap map[uint]int = map[uint]int{
	1: 1,
	2: 2,
	3: 4,
	4: 8,
	5: 16,
	6: 32,
	7: 64,
}

// Passed vocab should be skilled up and scheduled for practice according to the new level.
// Failed vocab should be skilled down and scheduled for practice tomorrow.
func (h *practiceHandler) post(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	check(err)

	requestData := make([]struct {
		ID     uint `json:"id"`
		Passed bool `json:"passed"`
	}, 0)
	err = json.Unmarshal(body, &requestData)
	check(err)

	for _, practiceItem := range requestData {
		var vocab *Vocab
		dbResult := h.db.First(&vocab, practiceItem.ID)
		check(dbResult.Error)

		if practiceItem.Passed {
			if vocab.KnowledgeLevel < maxKnowledge {
				vocab.KnowledgeLevel++
			}
			vocab.PracticeAt = inDays(knowledgeToPracticeMap[vocab.KnowledgeLevel])
		} else {
			if vocab.KnowledgeLevel > 0 {
				vocab.KnowledgeLevel--
			}
			vocab.PracticeAt = inDays(1)
		}

		dbResult = h.db.Save(&vocab)
		check(dbResult.Error)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
	return nil
}

type QueryParams struct {
	r *http.Request
}

func (q *QueryParams) Str(key, fallback string) string {
	s := q.r.URL.Query().Get(key)
	if s == "" {
		return fallback
	}
	return s
}

func (q *QueryParams) Int(key string, fallback int) int {
	s := q.r.URL.Query().Get(key)
	if s == "" {
		return fallback
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return i
}

func like(s string) string {
	return "%" + s + "%"
}

func inDays(n int) time.Time {
	t := time.Now()
	year, month, day := t.Date()
	return time.Date(year, month, day+n, 0, 0, 0, 0, t.Location())
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
