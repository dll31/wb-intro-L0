package server

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
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
}

func New() *Server {
	addr, exists := os.LookupEnv("SERVER_ADDR")
	if !exists {
		addr = ":8080"
	}

	page, exists := os.LookupEnv("SERVER_PAGE")
	if !exists {
		page = "index.html"
	}

	tmpl := template.Must(template.ParseFiles(page))

	return &Server{
		addr:   addr,
		page:   page,
		tmpl:   tmpl,
		server: http.Server{Addr: addr},
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

		order := Order{ID: id, Status: "processed"}
		jsonResponse, err := json.Marshal(order)
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
	http.ListenAndServe(s.addr, nil)
}

func (s *Server) Shutdown() {
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	log.Println("Graceful shutdown complete.")
}
