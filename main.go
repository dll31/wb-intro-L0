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
	natsstreaming "wb-intro-l0/nats-streaming"

	"github.com/joho/godotenv"
	stan "github.com/nats-io/stan.go"
)

var (
	message_counter int
	postgr          postgres.Postgres
	c               *cache.Cache
	server          *httpserver.Server
	ns              *natsstreaming.NatsStreaming
)

func init() {

	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func msgWorker(msg *stan.Msg) {
	message_counter++
	printMsg(msg, message_counter)
	var m model.Model
	if err := m.Unmarshal(&msg.Data); err != nil {
		log.Printf("Cannot unmarshal received data\n")
		return
	}

	if !m.ApplyIdFromFields() {
		return
	}

	err := postgr.Insert(&m)
	if err == nil {
		c.Set(m.Id, m.Fields)
	}
}

func main() {

	ns = natsstreaming.New()
	postgr.NewPool(postgres.NewDb())
	c = cache.New()

	//Cache recovering
	// ...

	server = httpserver.New(c)
	go server.Serve()

	ns.CreateConnection()
	ns.QSubscribe(msgWorker)

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			fmt.Printf("\nReceived an interrupt, unsubscribing and closing connection...\n\n")
			// Do not unsubscribe a durable on exit, except if asked to.
			server.Shutdown()
			ns.Shutdown()
			postgr.ClosePool()

			cleanupDone <- true
		}
	}()
	<-cleanupDone
}

func printMsg(m *stan.Msg, i int) {
	log.Printf("[#%d] Received: %v\n\n", i, m)
}
