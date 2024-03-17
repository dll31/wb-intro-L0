package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"wb-intro-l0/cache"
	postgres "wb-intro-l0/db/postgres"
	"wb-intro-l0/httpserver"
	"wb-intro-l0/model"

	"github.com/joho/godotenv"
	stan "github.com/nats-io/stan.go"
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {

	var (
		clusterID, clientID string
		URL                 string
		message_counter     int
		subj                string
		durable             string
		qgroup              string
	)

	var p postgres.Postgres
	p.NewPool(postgres.NewDb())
	defer p.ClosePool()

	c := cache.New()

	s := httpserver.New(c)
	go s.Serve()
	defer s.Shutdown()

	URL, exists := os.LookupEnv("URL")
	if !exists {
		URL = stan.DefaultNatsURL
	}

	clusterID, exists = os.LookupEnv("CLUSTER_ID")
	if !exists {
		clusterID = "test-cluster"
	}

	clientID, exists = os.LookupEnv("CLIENT_ID_SUB")
	if !exists {
		clientID = "sub"
	}

	subj, exists = os.LookupEnv("CHANNEL")
	if !exists {
		subj = "orders"
	}

	durable, exists = os.LookupEnv("DURABLE")
	if !exists {
		durable = "cluster-dur"
	}

	qgroup, exists = os.LookupEnv("Q_GROUP_NAME")
	if !exists {
		qgroup = "oders_group"
	}

	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(URL),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Fatalf("Connection lost, reason: %v", reason)
		}))
	if err != nil {
		log.Fatalf("Can't connect: %v.\nMake sure a NATS Streaming Server is running at: %s", err, URL)
	}
	log.Printf("Connected to %s clusterID: [%s] clientID: [%s]\n", URL, clusterID, clientID)

	mcb := func(msg *stan.Msg) {
		message_counter++
		printMsg(msg, message_counter)
		var m model.Model
		if err = m.Unmarshal(&msg.Data); err != nil {
			log.Printf("Cannot unmarshal received data\n")
			return
		}

		if !m.ApplyIdFromFields() {
			return
		}

		err := p.Insert(&m)
		if err == nil {
			c.Set(m.Id, m.Fields)
		}
	}

	sub, err := sc.QueueSubscribe(subj, qgroup, mcb, stan.DurableName(durable), stan.DeliverAllAvailable())
	if err != nil {
		sc.Close()
		log.Fatal(err)
	}

	log.Printf("Listening on [%s], clientID=[%s]\n", subj, clientID)

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			fmt.Printf("\nReceived an interrupt, unsubscribing and closing connection...\n\n")
			// Do not unsubscribe a durable on exit, except if asked to.
			if durable == "" {
				sub.Unsubscribe()
			}
			sc.Close()
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}

func printMsg(m *stan.Msg, i int) {
	log.Printf("[#%d] Received: %v\n\n", i, m)
}
