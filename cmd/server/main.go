package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	/*"net"*/

	/*	"net"*/
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/eyedeekay/i2pkeys"
	//sam
	"github.com/eyedeekay/onramp"
	"github.com/yuukanoo/rtchat/internal/handler"
	"github.com/yuukanoo/rtchat/internal/logging"
	"github.com/yuukanoo/rtchat/internal/service"
	"github.com/yuukanoo/rtchat/internal/turn"
)

func main() {
	e := flags{
		debug: flag.Bool("debug", false, "Should we launch in the debug mode?"),
		turn: turnFlags{
			realm:    flag.String("realm", "rtchat.io", "Realm used by the turn server."),
			publicIP: flag.String("turn-ip", "127.0.0.1", "IP Address that TURN can be contacted on. Should be publicly available."),
			port:     flag.Int("turn-port", 3478, "Listening port for the TURN/STUN endpoint."),
		},
		web: webFlags{
			port: flag.Int("http-port", 5000, "Web server listening port."),
		},
		i2p: i2pFlags{
			useI2P:  flag.Bool("i2p", false, "Should we use I2P?"),
			samIP:   flag.String("sam-ip", "127.0.0.1", "IP address on which the Simple Anonymous Messaging bridge can be reached"),
			samPort: flag.Int("sam-port", 7656, "Port on which the Simple Anonymous Messaging bridge can be reached"),
		},
		tls: tlsFlags{
			cert: flag.String("tls-cert", "cert.pem", "Path to the TLS certificate file."),
			key:  flag.String("tls-key", "keys.pem", "Path to the TLS key file."),
		},
	}

	flag.Parse()
	e.turn.i2p = e.i2p

	logger := logging.New(*e.debug)

	if *e.debug {
		logger.Info("launched in debug mode, extra output is expected")
	}

	// Instantiates the service that creates rooms
	service := service.New()

	// Instantiate and launch the turn server
	turnServer, err := turn.New(service, logger, &e.turn)

	if err != nil {
		log.Fatal(err)
	}

	defer turnServer.Close()

	// Instantiate the application router
	r, err := handler.New(service, logger, &e.turn)

	if err != nil {
		log.Fatal(err)
	}

	defer r.Close()
	//var l net.Listener
	//var garlic *onramp.Garlic
	garlic, err := onramp.NewGarlic("rtchat", e.turn.SAMAddress(), []string{})
	if err != nil {
		log.Fatal(err)
	}
	l, err := garlic.ListenTLS()
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	e.web.host = l.Addr().(i2pkeys.I2PAddr).Base32()
	// Launch the HTTP server!

	s := &http.Server{Handler: r.Handler(), Addr: l.Addr().(i2pkeys.I2PAddr).Base32()}

	go func() {

		if err := s.Serve(l); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	defer s.Close()

	logger.Info(`HTTPS server launched:
	Listening:	https://%s`, e.web.Address())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	logger.Info("Shutting down, goodbye 👋")
}

// flags represents options which can be passed to internal packages.
type flags struct {
	debug *bool
	turn  turnFlags
	web   webFlags
	i2p   i2pFlags
	tls   tlsFlags
}

// turnFlags contains turn server related configuration.
type turnFlags struct {
	realm    *string
	publicIP *string
	port     *int
	i2p      i2pFlags
}

// webFlags contains web specific flags.
type webFlags struct {
	port *int
	host string
}

type i2pFlags struct {
	useI2P  *bool
	samIP   *string
	samPort *int
}

type tlsFlags struct {
	cert *string
	key  *string
}

func (f *turnFlags) Realm() string      { return *f.realm }
func (f *turnFlags) PublicIP() net.IP   { return net.ParseIP(*f.publicIP) }
func (f *turnFlags) Port() int          { return *f.port }
func (f *turnFlags) TurnURL() string    { return fmt.Sprintf("turn:%s:%d", *f.publicIP, *f.port) }
func (f *turnFlags) StunURL() string    { return fmt.Sprintf("stun:%s:%d", *f.publicIP, *f.port) }
func (f *webFlags) Address() string     { return fmt.Sprintf("%s:%d", f.host, *f.port) }
func (f *turnFlags) SAMAddress() string { return fmt.Sprintf("%s:%d", *f.i2p.samIP, *f.i2p.samPort) }
