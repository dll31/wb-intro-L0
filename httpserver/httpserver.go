package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
	"wb-intro-l0/cache"
)

type Order struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Server struct {
	addr   string
	page   string
	tmpl   *template.Template
	server http.Server
	c      *cache.Cache
}

func New(c *cache.Cache) *Server {
	addr, exists := os.LookupEnv("SERVER_ADDR")
	if !exists {
		addr = ":8080"
	}

	page, exists := os.LookupEnv("SERVER_PAGE")
	if !exists {
		page = "httpserver/index.html"
	}

	tmpl := template.Must(template.ParseFiles(page))

	return &Server{
		addr:   addr,
		page:   page,
		tmpl:   tmpl,
		server: http.Server{Addr: addr},
		c:      c,
	}
}

func (s *Server) executeTemplate(w *http.ResponseWriter, data any) {
	(*w).Header().Set("Content-Type", "text/html")
	if err := s.tmpl.Execute(*w, data); err != nil {
		http.Error(*w, err.Error(), http.StatusInternalServerError)
		log.Printf("Execute template error: %+v", err)
	}
}

func (s *Server) RequestHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "POST":
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id := r.Form.Get("id")
		fields := s.c.Get(id)

		jsonResponse, err := json.Marshal(fields)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var buff bytes.Buffer
		err = json.Indent(&buff, jsonResponse, "", "\t")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.executeTemplate(&w, buff.String())

	default:
		s.executeTemplate(&w, nil)
	}
}

func (s *Server) Serve() {
	http.HandleFunc("/", s.RequestHandler)
	if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func (s *Server) Shutdown() {
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	log.Println("Graceful shutdown complete.")
}
