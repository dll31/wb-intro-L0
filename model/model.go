package model

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
)

type Model struct {
	Id     string                 `json:"order_uid"`
	Fields map[string]interface{} `json:"-"`
}

func (m *Model) Init() {
	// rand.Seed(time.Now().UnixNano())
}

func ReadFromFile(filename string) (body []byte, err error) {
	body, err = os.ReadFile(filename)
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
		body = make([]byte, 0)
	}

	return
}

func (m *Model) Unmarshal(body *[]byte) (err error) {
	if err = json.Unmarshal(*body, &m.Fields); err != nil {
		return
	}

	if n, ok := m.Fields["order_uid"].(string); ok {
		m.Id = string(n)
	}

	return
}

func MakeRandomId() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	n := 19
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (m *Model) ApplyIdFromFields() (ok bool) {
	n, ok := m.Fields["order_uid"].(string)
	if !ok {
		log.Printf("Cannot apply id\n")
		return
	}
	m.Id = string(n)
	delete(m.Fields, "order_uid")
	return
}

func (m *Model) FillId(id string) {
	m.Id = id
	m.Fields["order_uid"] = id
}

func (m *Model) ToBytes() (buffer []byte) {
	buffer, err := json.Marshal(m.Fields)
	if err != nil {
		fmt.Println("Cannot marshal model")
	}

	return
}
