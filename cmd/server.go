package server

import (
	"fmt"
	"log"
	"net"
	"time"

	/*"net"*/

	/*	"net"*/
	"net/http"

	"github.com/go-i2p/i2pkeys"
	//sam
	"github.com/go-i2p/onramp"
	"github.com/yuukanoo/rtchat/internal/handler"
	"github.com/yuukanoo/rtchat/internal/logging"
	"github.com/yuukanoo/rtchat/internal/service"
	"github.com/yuukanoo/rtchat/internal/turn"
)

var garlic *onramp.Garlic
var l net.Listener
var s *http.Server
var r handler.Router
var turnServer turn.Server
var serv service.Service
var logger logging.Logger
var err error

func Serve(e Flags, appname string) string {
	logger = logging.New(false)

	// Instantiates the service that creates rooms
	serv = service.New()

	// Instantiate and launch the turn server
	turnServer, err = turn.New(serv, logger, &e.Turn)

	if err != nil {
		log.Fatal(err)
	}

	//defer turnServer.Close()

	// Instantiate the application router
	r, err = handler.New(serv, logger, &e.Turn)

	if err != nil {
		log.Fatal(err)
	}

	//defer r.Close()
	garlic, err = onramp.NewGarlic(appname, e.Turn.SAMAddress(),
		[]string{"inbound.length=1", "outbound.length=1",
			"inbound.backupQuantity=2", "outbound.backupQuantity=2",
			"inbound.quantity=3", "outbound.quantity=3"},
	)
	if err != nil {
		log.Fatal(err)
	}
	l, err = garlic.ListenTLS()
	if err != nil {
		log.Fatal(err)
	}
	//defer l.Close()
	e.Web.Host = l.Addr().(i2pkeys.I2PAddr).Base32()
	// Launch the HTTP server!

	s = &http.Server{
		Handler:      r.Handler(),
		Addr:         l.Addr().(i2pkeys.I2PAddr).Base32(),
		ReadTimeout:  50 * time.Second,
		WriteTimeout: 100 * time.Second,
	}

	go func() {
		if err := s.Serve(l); err != http.ErrServerClosed {
			log.Fatal(err)
		}
		log.Println("Server closed")
	}()

	logger.Info(`HTTPS server launched:
	Listening:	https://%s`, e.Web.Address())

	return fmt.Sprintf("https://%s", l.Addr().(i2pkeys.I2PAddr).Base32())
}

func Close() {
	garlic.Close()
	s.Close()
	l.Close()
	r.Close()
	turnServer.Close()
}

// Flags represents options which can be passed to internal packages.
type Flags struct {
	//debug *bool
	Turn TurnFlags
	Web  WebFlags
	//I2p  I2pFlags
	/*	tls   tlsFlags*/
}

// TurnFlags contains turn server related configuration.
type TurnFlags struct {
	RealmString    *string
	PublicIPString *string
	PortInt        *int
	I2p            I2pFlags
}

// WebFlags contains web specific flags.
type WebFlags struct {
	Port *int
	Host string
}

type I2pFlags struct {
	SamIP   *string
	SamPort *int
}

func (f *TurnFlags) Realm() string    { return *f.RealmString }
func (f *TurnFlags) PublicIP() net.IP { return net.ParseIP(*f.PublicIPString) }
func (f *TurnFlags) Port() int        { return *f.PortInt }
func (f *TurnFlags) TurnURL() string {
	return fmt.Sprintf("turn:%s:%d", *f.PublicIPString, *f.PortInt)
}
func (f *TurnFlags) StunURL() string {
	return fmt.Sprintf("stun:%s:%d", *f.PublicIPString, *f.PortInt)
}
func (f *WebFlags) Address() string     { return fmt.Sprintf("%s:%d", f.Host, *f.Port) }
func (f *TurnFlags) SAMAddress() string { return fmt.Sprintf("%s:%d", *f.I2p.SamIP, *f.I2p.SamPort) }
