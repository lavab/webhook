package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/bitly/go-nsq"
	r "github.com/dancannon/gorethink"
	"github.com/lavab/api/models"
	"github.com/lavab/webhook/events"
	"github.com/namsral/flag"
)

var (
	rethinkAddress  = flag.String("rethinkdb_address", "127.0.0.1:28015", "Address of the RethinkDB database")
	rethinkDatabase = flag.String("rethinkdb_db", "dev", "Database name on the RethinkDB server")
	lookupdAddress  = flag.String("lookupd_address", "127.0.0.1:4161", "Address of the lookupd server")
)

func main() {
	flag.Parse()

	session, err := r.Connect(r.ConnectOpts{
		Address:  *rethinkAddress,
		Database: *rethinkDatabase,
	})
	if err != nil {
		log.Fatal(err)
	}

	incoming, err := nsq.NewConsumer("hook_incoming", "handler", nsq.NewConfig())
	if err != nil {
		log.Fatal(err)
	}

	incoming.SetLogger(&silentLogger{}, nsq.LogLevelDebug)

	incoming.AddConcurrentHandlers(nsq.HandlerFunc(func(msg *nsq.Message) error {
		var event *events.Incoming
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Print(err)
			return err
		}

		cursor, err := r.Table("webhooks").GetAllByIndex("targetType", []interface{}{
			event.Account,
			"incoming",
		}).Run(session)
		if err != nil {
			log.Print(err)
			return err
		}
		defer cursor.Close()
		var hooks []*models.Webhook
		if err := cursor.All(&hooks); err != nil {
			log.Print(err)
			return err
		}

		var wg sync.WaitGroup
		wg.Add(len(hooks))

		for _, hook := range hooks {
			go func() {
				defer func() {
					wg.Done()
					msg.Touch()
				}()

				resp, err := http.Post(hook.Address, "application/json", bytes.NewReader(msg.Body))
				if err != nil {
					log.Print(err)
					return
				}

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Print(err)
					return
				}

				log.Printf("Incoming hook to %s - %d: %s", hook.Address, resp.StatusCode, string(body))
			}()
		}

		wg.Wait()

		return nil
	}), 10)

	onboarding, err := nsq.NewConsumer("hook_onboarding", "handler", nsq.NewConfig())
	if err != nil {
		log.Fatal(err)
	}

	onboarding.SetLogger(&silentLogger{}, nsq.LogLevelDebug)

	onboarding.AddConcurrentHandlers(nsq.HandlerFunc(func(msg *nsq.Message) error {
		var event *events.Onboarding
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Print(err)
			return err
		}

		cursor, err := r.Table("webhooks").GetAllByIndex("type", "onboarding").Run(session)
		if err != nil {
			log.Print(err)
			return err
		}
		defer cursor.Close()
		var hooks []*models.Webhook
		if err := cursor.All(&hooks); err != nil {
			log.Print(err)
			return err
		}

		var wg sync.WaitGroup
		wg.Add(len(hooks))

		for _, hook := range hooks {
			go func() {
				defer func() {
					wg.Done()
					msg.Touch()
				}()

				resp, err := http.Post(hook.Address, "application/json", bytes.NewReader(msg.Body))
				if err != nil {
					log.Print(err)
					return
				}

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Print(err)
					return
				}

				log.Printf("Onboarding hook to %s - %d: %s", hook.Address, resp.StatusCode, string(body))
			}()
		}

		wg.Wait()

		return nil
	}), 10)

	if err := onboarding.ConnectToNSQLookupd(*lookupdAddress); err != nil {
		log.Fatal(err)
	}

	log.Print("Started all handlers")

	select {}
}
