package natsstreaming

import (
	"log"
	"os"

	"github.com/nats-io/stan.go"
)

type NatsStreaming struct {
	clusterID, clientID string
	URL                 string
	message_counter     int
	subj                string
	durable             string
	qgroup              string
	conn                stan.Conn
	sub                 stan.Subscription
}

func New() *NatsStreaming {
	URL, exists := os.LookupEnv("URL")
	if !exists {
		URL = stan.DefaultNatsURL
	}

	clusterID, exists := os.LookupEnv("CLUSTER_ID")
	if !exists {
		clusterID = "test-cluster"
	}

	clientID, exists := os.LookupEnv("CLIENT_ID_SUB")
	if !exists {
		clientID = "sub"
	}

	subj, exists := os.LookupEnv("CHANNEL")
	if !exists {
		subj = "orders"
	}

	durable, exists := os.LookupEnv("DURABLE")
	if !exists {
		durable = "cluster-dur"
	}

	qgroup, exists := os.LookupEnv("Q_GROUP_NAME")
	if !exists {
		qgroup = "oders_group"
	}

	return &NatsStreaming{
		clusterID:       clusterID,
		clientID:        clientID,
		URL:             URL,
		subj:            subj,
		durable:         durable,
		qgroup:          qgroup,
		message_counter: 0,
	}
}

func (ns *NatsStreaming) CreateConnection() error {
	var err error
	ns.conn, err = stan.Connect(ns.clusterID, ns.clientID, stan.NatsURL(ns.URL), stan.Pings(2, 2), stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
		log.Fatalf("Connection lost, reason: %v", reason)
	}))
	if err != nil {
		log.Fatalf("Can't connect: %v.\nMake sure a NATS Streaming Server is running at: %s", err, ns.URL)
		return err
	}
	log.Printf("Connected to %s clusterID: [%s] clientID: [%s]\n", ns.URL, ns.clusterID, ns.clientID)
	return err
}

func (ns *NatsStreaming) QSubscribe(messageWorker func(msg *stan.Msg)) {
	var err error
	ns.sub, err = ns.conn.QueueSubscribe(ns.subj, ns.qgroup, messageWorker, stan.DurableName(ns.durable), stan.DeliverAllAvailable())
	if err != nil {
		ns.conn.Close()
		log.Fatal(err)
		return
	}
	log.Printf("Listening on [%s], clientID=[%s]\n", ns.subj, ns.clientID)
}

func (ns *NatsStreaming) Shutdown() {
	if ns.durable == "" {
		ns.sub.Unsubscribe()
	}
	ns.conn.Close()
}
