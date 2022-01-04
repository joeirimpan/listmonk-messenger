package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/joeirimpan/listmonk-messenger/messenger"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	flag "github.com/spf13/pflag"
)

var (
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	ko     = koanf.New(".")

	// Version of the build injected at build time.
	buildString = "unknown"
)

type MessengerCfg struct {
	Config string `koanf:"config"`
}

type App struct {
	messengers map[string]messenger.Messenger
}

func init() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}
	f.StringSlice("config", []string{"config.toml"},
		"Path to one or more TOML config files to load in order")
	f.StringSlice("msgr", []string{"pinpoint"},
		"Name of messenger. Can specify multiple values.")
	f.Bool("version", false, "Show build version")
	if err := f.Parse(os.Args[1:]); err != nil {
		log.Fatalf("error parsing flags: %v", err)
	}

	// Display version.
	if ok, _ := f.GetBool("version"); ok {
		fmt.Println(buildString)
		os.Exit(0)
	}

	// Read the config files.
	cFiles, _ := f.GetStringSlice("config")
	for _, f := range cFiles {
		log.Printf("reading config: %s", f)
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			log.Printf("error reading config: %v", err)
		}
	}

	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		log.Fatalf("error loading flags: %v", err)
	}
}

// loadMessengers loads all messages mentioned in posflag into application.
func loadMessengers(msgrs []string, app *App) {
	app.messengers = make(map[string]messenger.Messenger)

	for _, m := range msgrs {
		var cfg MessengerCfg
		if err := ko.Unmarshal("messenger."+m, &cfg); err != nil {
			log.Fatalf("error reading %s messenger config: %v", m, err)
		}

		var (
			msgr messenger.Messenger
			err  error
		)
		switch m {
		case "pinpoint":
			msgr, err = messenger.NewPinpoint([]byte(cfg.Config))
		default:
			log.Fatalf("invalid provider: %s", m)
		}

		if err != nil {
			log.Fatalf("error creating %s messenger: %v", m, err)
		}

		app.messengers[m] = msgr
		log.Printf("loaded %s\n", m)
	}
}

func main() {
	// load messengers
	app := &App{}

	loadMessengers(ko.Strings("msgr"), app)

	r := chi.NewRouter()
	r.Post("/webhook/{provider}", wrap(app, handlePostback))

	// HTTP Server.
	srv := &http.Server{
		Addr:         ko.String("server.address"),
		ReadTimeout:  ko.Duration("server.read_timeout"),
		WriteTimeout: ko.Duration("server.write_timeout"),
		Handler:      r,
	}

	logger.Printf("starting on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatalf("couldn't start server: %v", err)
	}
}
