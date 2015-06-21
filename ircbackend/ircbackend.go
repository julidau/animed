package main

import (
	"encoding/json"
	"flag"

	"net/http"

	"strconv"
	"strings"

	"errors"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"fmt"
	"os"

	_ "github.com/thoj/go-ircevent"
)

var (
	configpath = flag.String("config", "default.json", "The JSON Config file for the backend")
	httpstatus = flag.Bool("enableStatus", true, "listens on via --listen given address and serves status")
	httplisten = flag.String("listen", "localhost:8081", "where, if enabled, the http server should listen on")
)

type configs []IRCconfig

type IRCconfig struct {
	Host     string
	Username string
	Nickname string
	Timeout  string
}

type Backend struct {
	bots []*bot
}

type Frontend struct {
	backend *Backend
	in      io.Reader
	out     io.Writer
}

func NewBackend(config []byte) (*Backend, error) {
	b := Backend{}
	if err := b.ReadConfig(config); err != nil {
		return nil, err
	}

	return &b, nil
}

func (b *Backend) ReadConfig(file []byte) (err error) {
	// read config from file
	var (
		configs []IRCconfig
	)

	if err = json.Unmarshal(file, &configs); err != nil {
		return err
	}

	// log Config
	log.Println(configs)

	// start ircbots
	for i := range configs {
		config := configs[i]
		bot := NewBot(config.Nickname, config.Username)

		if bot.Timeout, err = time.ParseDuration(config.Timeout); err != nil {
			bot.Timeout, _ = time.ParseDuration("1s")
		}

		if err = bot.Connect(config.Host); err != nil {
			log.Println("bot error during connect: ", err)
			continue
		}

		log.Println("nick: ", bot.GetNick())

		b.bots = append(b.bots, bot)
		go bot.Loop()
	}

	if len(b.bots) == 0 {
		return errors.New("no bots configured")
	}

	return nil
}

func (b *Frontend) Run() {
	if b.in == nil || b.out == nil {
		return
	}

	var (
		line string
		err  error
	)
	
	for {
		n, err := fmt.Fscanln(b.in, &line)

		if err != nil || n == 0 {
			break
		}
		
		fmt.Fprintln(b.out, line);
		continue

		request := strings.SplitN(line, " ", 3)

		if len(request) < 2 {
			if _, err := b.out.Write([]byte("not enough args for " + line)); err != nil {
				return
			}

			continue
		}

		switch strings.ToLower(request[0]) {
		case "downloads":
			for i := range b.backend.bots {
				bot := b.backend.bots[i]
				for j := range bot.downloads {
					if _, err := b.out.Write([]byte(j)); err != nil {
						return
					}
				}
			}
			break
		}
	}

	if err != nil && err != io.EOF {
		log.Println(err)
	}
}

func main() {
	flag.Parse()

	// load config
	config, err := ioutil.ReadFile(filepath.Join(*configpath))
	if err != nil {
		log.Fatal(err)
	}

	backend, err := NewBackend(config)

	if err != nil {
		log.Fatal(err)
	}

	if *httpstatus {
		StatusHandler := func(w http.ResponseWriter, req *http.Request) {
			var response string
			response = "<html><meta char-encoding=\"utf8\"/><meta http-equiv=\"refresh\" content=\"2\">"
			for i := range backend.bots {
				bot := backend.bots[i]
				for j := range bot.downloads {
					progress := float32(bot.downloads[j].current)
					progress /= float32(bot.downloads[j].size)
					progress *= 100
					response += j + " " + strconv.FormatFloat(float64(progress), 'f', 0, 32) + "%<br/>"
				}
			}
			response += "</html>"

			if _, err := w.Write([]byte(response)); err != nil {
				return
			}
		}

		http.HandleFunc("/status", StatusHandler)
		http.ListenAndServe(*httplisten, nil)
	}

	for {	
		handler := Frontend{backend, os.Stdin, os.Stdout}
		handler.Run()
	}
}
