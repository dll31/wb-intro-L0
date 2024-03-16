package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
)

type Order struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

var tmpl *template.Template

func main() {

	tmpl = template.Must(template.ParseFiles("index.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
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

			w.Header().Set("Content-Type", "text/html")
			if err := tmpl.Execute(w, buff.String()); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	http.ListenAndServe(":8080", nil)
}
