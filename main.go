// Package main initializes and runs the server for the OwlDB project.
// It sets up the necessary dependencies, including the document schema, database,
// and authentication services.
package main

import (
	"flag"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/auth"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/concurrentSkipList"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/logger"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/patcher"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/resourceCreatorService"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/resourceDeleterService"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/resourceGetterService"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/resourcePatcherService"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/subscriptionManager"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/validation"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/db"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/document"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/server"
)

func main() {
	var srv http.Server
	var port int
	var schema string
	var tokens string
	var err error

	// Parse command-line flags for port, schema, and tokens
	flag.IntVar(&port, "p", 3318, "port number")

	// might need to change "document.json" to "" to make sure when passing in schema exist

	flag.StringVar(&schema, "s", "", "document schema")

	flag.StringVar(&tokens, "t", "", "tokens")
	flag.Parse()

	// Initialize logging options
	logOpts := &logger.PrettyHandlerOptions{
		Level:    slog.LevelDebug,
		Colorize: true,
	}
	handler1 := logger.NewPrettyHandler(os.Stdout, logOpts)
	logger1 := slog.New(handler1)
	slog.SetDefault(logger1)

	//Check if schema file is provided
	if schema == "" {
		fmt.Printf("Error: Schema file not specified. Use -s <schema filename>\n")
		os.Exit(1)
	}
	validator, err := validation.New(schema)

	if err != nil {
		fmt.Printf("Error: Bad schema file\n")
		os.Exit(1)
	}
	// Dependency injection and factory initialization
	var docColFactory document.DocumentIndexFactory[document.DocumentIndex[string, *document.Collection]]

	var dbFactory resourceCreatorService.DBFactory[*db.Database[string, *document.Document]]

	var docFactory db.DocFactory[*document.Document]
	var newerColFactory document.CollectionFactory
	newerColFactory = func(colName string) *document.Collection {
		return &document.Collection{
			Name:                colName,
			Docs:                concurrentSkipList.NewSL[string, *document.Document](string(rune(0)), string(rune(127))),
			SubscriptionManager: subscriptionManager.NewColSubManager(concurrentSkipList.NewSL[string, subscriptionManager.Colsubscriber](string(rune(0)), string(rune(127)))),
		}
	}

	var smFactory document.SubscriptionManagerFactory = func() document.SubscriptionManager {
		subs := concurrentSkipList.NewSL[string, *chan []byte](string(rune(0)), string(rune(127)))
		return subscriptionManager.New(subs)
	}

	var idtosubfactory subscriptionManager.IdToSubFactory = func() subscriptionManager.IdToSub[string, *chan []byte] {
		return concurrentSkipList.NewSL[string, *chan []byte](string(rune(0)), string(rune(127)))
	}

	docSubs := concurrentSkipList.NewSL[string, *subscriptionManager.SubscriptionManager](string(rune(0)), string(rune(127)))
	messager := subscriptionManager.NewMessager(idtosubfactory, docSubs)
	docFactory = func(payload []byte, user string, path string) *document.Document {

		newDoc := document.New(payload, user, path, docColFactory, newerColFactory, smFactory, validator, patcher.Patcher{}, messager)
		return newDoc
	}

	docColFactory = func() document.DocumentIndex[string, *document.Collection] {
		newCollections := concurrentSkipList.NewSL[string, *document.Collection](string(rune(0)), string(rune(127)))
		return newCollections
	}

	dbFactory = func(name string) *db.Database[string, *document.Document] {
		newDBIndices := concurrentSkipList.NewSL[string, *document.Document](string(rune(0)), string(rune(127)))
		sm := subscriptionManager.NewColSubManager(concurrentSkipList.NewSL[string, subscriptionManager.Colsubscriber](string(rune(0)), string(rune(127))))
		return db.New[string, *document.Document](name, docFactory, newDBIndices, sm, validator)

	}
	//FOR CRUD OPERATIONS
	// Initialize database and resource services
	dbs := concurrentSkipList.NewSL[string, *db.Database[string, *document.Document]](string(rune(0)), string(rune(127)))
	var rcsDB resourceCreatorService.DatabaseIndex[string, *db.Database[string, *document.Document]] = dbs
	var rgsDB resourceGetterService.DatabaseIndex[string, *db.Database[string, *document.Document]] = dbs
	var rdsDB resourceDeleterService.DatabaseIndex[string, *db.Database[string, *document.Document]] = dbs
	var rpsDB resourcePatcherService.DatabaseIndex[string, *db.Database[string, *document.Document]] = dbs
	rcs := resourceCreatorService.New(rcsDB, dbFactory, validator)
	rgs := resourceGetterService.New(rgsDB)
	rds := resourceDeleterService.New(rdsDB)
	rps := resourcePatcherService.New(rpsDB)

	// Initialize authentication services

	var tokenMap auth.TokenIndex[string, auth.Session] = concurrentSkipList.NewSL[string, auth.Session](string(rune(0)), string(rune(127)))

	authService := auth.New(tokenMap, tokens)

	// Initialize the server handler
	handler := server.New(rds, rgs, rcs, authService, rps)
	srv.Handler = handler
	srv.Addr = fmt.Sprintf(":%d", port)

	// The following code should go last and remain unchanged.
	// Note that you must actually initialize 'server' and 'port'
	// before this.  Note that the server is started below by
	// calling ListenAndServe.  You must not start the server
	// before this.

	// signal.Notify requires the channel to be buffered
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt, syscall.SIGTERM)
	go func() {
		// Wait for Ctrl-C signal
		<-ctrlc
		srv.Close()
	}()

	// Start server
	slog.Info("Listening", "port", port)
	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		slog.Error("Server closed", "error", err)
	} else {
		slog.Info("Server closed", "error", err)
	}

	slog.Info("Server closed")
}
