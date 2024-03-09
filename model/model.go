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
	// fmt.Println(string(body))
	return
}

func (m *Model) Unmarshal(body *[]byte) (err error) {
	if err = json.Unmarshal(*body, &m.Fields); err != nil {
		log.Panic(err)
		return
	}

	if n, ok := m.Fields["order_uid"].(string); ok {
		m.Id = string(n)
	}

	delete(m.Fields, "order_uid")

	// fmt.Printf("%+v", m)
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

func (m *Model) FillId(id string) {
	m.Id = id
}

func (m *Model) ToBytes() (buffer []byte) {
	buffer = []byte(fmt.Sprintf("%v", m))
	return
}
