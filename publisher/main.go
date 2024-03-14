package main

import (
	"log"
	"os"
	"wb-intro-l0/model"

	"github.com/joho/godotenv"
	stan "github.com/nats-io/stan.go"
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load("../.env"); err != nil {
		log.Print("No .env file found")
	}
}

func main() {

	var (
		clusterID, clientID string
		URL                 string
		subj                string
	)

	URL, exists := os.LookupEnv("URL")
	if !exists {
		URL = stan.DefaultNatsURL
	}

	clusterID, exists = os.LookupEnv("CLUSTER_ID")
	if !exists {
		clusterID = "test-cluster"
	}

	clientID, exists = os.LookupEnv("CLIENT_ID_PUB")
	if !exists {
		clientID = "pub"
	}

	subj, exists = os.LookupEnv("CHANNEL")
	if !exists {
		subj = "orders"
	}
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(URL))
	if err != nil {
		log.Fatal("Stan connection error:", err)
	}
	defer sc.Close()

	var m model.Model
	m.Init()

	body, err := model.ReadFromFile("../model.json")
	if err != nil {
		return
	}
	if err = m.Unmarshal(&body); err != nil {
		return
	}

	m.FillId(model.MakeRandomId())
	// fmt.Printf("Send message with data: %v", m)

	sc.Publish(subj, m.ToBytes())

}
