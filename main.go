package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
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
	env             map[string]string
)

func init() {

	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	var err error
	env, err = godotenv.Read(".env")
	if err != nil {
		log.Print("No .env file found")
	}
}

func msgWorker(msg *stan.Msg) {
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
		message_counter++
		env["CACHE_LEN"] = strconv.Itoa(message_counter)
		err = godotenv.Write(env, ".env")
		if err != nil {
			log.Println("Cannot write update in env file")
		}
	}
}

func main() {

	ns = natsstreaming.New()
	postgr.NewPool(postgres.NewDb())
	c = cache.New()

	//Cache recovering
	cS := env["CORRECT_SHUTDOWN"]
	if cS == "" {
		cS = "false"
		env["CORRECT_SHUTDOWN"] = cS
	}

	correctShutdown, err := strconv.ParseBool(cS)
	if err != nil {
		correctShutdown = false
	}

	if !correctShutdown {
		log.Println("Last shutdown was uncorrect. Trying to recover cache")
		cacheLenStr := env["CACHE_LEN"]
		if cacheLenStr == "" {
			log.Println("Cannot get 'CACHE_LEN' from env. CACHE_LEN=0. Program will start as usually")
			cacheLenStr = "0"
			env["CACHE_LEN"] = cacheLenStr
		} else {
			cacheLen, err := strconv.ParseInt(cacheLenStr, 10, 0)
			if err != nil {
				log.Println("Cannot parse 'CACHE_LEN'. CACHE_LEN=0. Continue")
				env["CACHE_LEN"] = "0"
			} else {
				recCache := postgr.SelectLastNRows(int(cacheLen))
				for key, value := range recCache {
					c.Set(key, value)
				}
			}

		}

	}
	env["CORRECT_SHUTDOWN"] = "false"
	env["CACHE_LEN"] = "0"
	err = godotenv.Write(env, ".env")
	if err != nil {
		log.Println("Cannot write update in env file")
	}

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

			env["CACHE_LEN"] = "0"
			env["CORRECT_SHUTDOWN"] = "true"
			err = godotenv.Write(env, ".env")
			if err != nil {
				log.Println("Cannot write update in env file")
			}

			cleanupDone <- true
		}
	}()
	<-cleanupDone
}

func printMsg(m *stan.Msg, i int) {
	log.Printf("[#%d] Received: %v\n\n", i, m)
}
