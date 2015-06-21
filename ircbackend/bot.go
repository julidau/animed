package main

import (
	"errors"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"net"
	"os"
	"sync"

	irc "github.com/julidau/go-ircevent"
)

type bot struct {
	*irc.Connection

	// mutex protects downloads map
	downMutex sync.Mutex
	downloads map[string]*DCCDownloader
}

// rewrite all # to random numbers
// effectivly creating unique usernames
func rewriteNickname(nickname string) string {
	
	str := []byte(nickname)
	
	for c := range str {
		if str[c] == '#' {
			str[c] = byte((rand.Int()%10)+'0')
		}
	}

	return nickname
}

func NewBot(ircnick string, ircusername string) *bot {
	rand.Seed(int64(time.Now().Nanosecond()))
	temp := bot{irc.IRC(rewriteNickname(ircnick), ircusername), sync.Mutex{}, make(map[string]*DCCDownloader)}
	temp.AddCallback("CTCP", temp.CTCPCallback)
	return &temp
}

func (this *bot) CTCPCallback(evt *irc.Event) {
	this.Log.Println("DCC request: ", evt.Raw, evt.Arguments)

	// little helper function to extract the next argument
	// from the event
	nextArg := func(array []string, i *int) (string, error) {
		*i += 1

		if *i >= len(array) {
			return "", errors.New("errogenous request")
		}

		return array[*i], nil
	}

	request := strings.Split(evt.Arguments[1], " ")
	this.Log.Println(request)

	if len(request) < 5 {
		return
	}

	currentArg := new(int)
	*currentArg = -1

	if arg, err := nextArg(request, currentArg); strings.ToUpper(arg) != "DCC" || err != nil {
		this.Log.Println("arg: ", arg)
		return
	}

	this.Log.Println("looking for send or accept")

	switch arg, _ := nextArg(request, currentArg); arg {
	case "SEND":
		var (
			filename string
			ip       uint32
			port     uint16
			size     uint64

			arg string
			err error
		)
		// DCC SEND filename ip port [filesize]
		if arg, err = nextArg(request, currentArg); arg[0] == '"' {
			// find closing \"
			for ; err == nil; arg, err = nextArg(request, currentArg) {
				filename += arg

				if arg[len(arg)-1] == '"' {
					break
				}
			}

			// join all arguments
			if len(filename) < 2 || err != nil {
				return
			}

			filename = filename[1 : len(filename)-2]
		} else {
			filename = arg
		}

		this.Log.Println(filename)

		// i+1 => ip
		if arg, err = nextArg(request, currentArg); err != nil {
			return
		}

		if temp, err := strconv.ParseUint(arg, 10, 32); err != nil {
			return
		} else {
			ip = uint32(temp)
		}

		// i+2 => port
		if arg, err = nextArg(request, currentArg); err != nil {
			return
		}

		if temp, err := strconv.ParseUint(arg, 10, 16); err != nil {
			return
		} else {
			port = uint16(temp)
		}

		// i+3 => size
		arg, _ = nextArg(request, currentArg)

		if temp, err := strconv.ParseUint(arg, 10, 64); err != nil {
			size = math.MaxUint64
		} else {
			size = temp
		}

		// everything is assembled, start the Download
		downloader := DCCDownloader{size: size}

		var addr net.TCPAddr
		addr.IP = net.IPv4(uint8(ip>>24), uint8(ip>>16), uint8(ip>>8), uint8(ip))
		addr.Port = int(port)

		socket, err := net.DialTCP("tcp4", nil, &addr)
		if err != nil {
			return
		}

		file, err := os.Create(filename)
		if err != nil {
			return
		}
		// ressources were aquired,
		// add downloader to bot
		this.downloads[filename] = &downloader

		// start download
		go func(this *bot) {
			defer socket.Close()
			defer file.Close()

			if err := downloader.Download(socket, file); err != nil {
				this.Log.Println(err)
			}

			// remove downloader from map
			this.downMutex.Lock()
			delete(this.downloads, file.Name())
			this.downMutex.Unlock()

			this.Log.Println("download done:", file.Name())
		}(this)

		break
	case "ACCEPT":
		// DCC ACCEPT filename port filesize
		break
	}
}

func (this *bot) ConnectAsync(host string, connected chan<- *bot) {
	switch err := this.Connection.Connect(host); err {
	case nil:
		connected <- this
		return
	default:
		this.Error = make(chan error, 1)
		this.Error <- err
	}
}
