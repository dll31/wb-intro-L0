package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"wb-intro-l0/model"

	"github.com/jackc/pgx/v5/pgxpool"
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

	username, _ := os.LookupEnv("POSTGRES_USER")
	password, _ := os.LookupEnv("PGPASSWORD")
	host, _ := os.LookupEnv("POSTGRES_HOST")
	port, _ := os.LookupEnv("POSTGRES_PORT")
	dbName, _ := os.LookupEnv("POSTGRES_DB")

	connString := "postgres://" + username + ":" + password + "@" + host + ":" + port + "/" + dbName

	dbpool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()

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

		insertIntoDb(dbpool, dbName, &m)

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

func insertIntoDb(conn *pgxpool.Pool, dbName string, m *model.Model) {
	insertString := fmt.Sprintf("INSERT INTO %s (order_uid, order_data) VALUES ", dbName)
	jsonFields, err := json.Marshal(m.Fields)
	if err != nil {
		log.Println("Cannot marshal fields in insert")
		return
	}
	_, err = conn.Exec(context.Background(), (insertString + "($1, $2)"), m.Id, jsonFields)

	if err != nil {
		log.Printf("Cannot insert into db. Error: %s\n", err)
	}
}

// func selectFromDb(conn *pgxpool.Pool, m *model.Model) {
// 	selectString := "SELECT order_uid, order_data FROM "
// 	row := conn.QueryRow(context.Background(), selectString)
// 	var jsonFields []byte
// 	row.Scan(m.Id, jsonFields)
// 	err := m.Unmarshal(&jsonFields)
// 	if err != nil {
// 		log.Println("Cannot unmarshal in select from db")
// 	}
// }
