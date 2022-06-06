package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/eyedeekay/i2pkeys"
	sam "github.com/eyedeekay/sam3/helper"
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
			publicIP: flag.String("turn-ip", "192.168.0.14", "IP Address that TURN can be contacted on. Should be publicly available."),
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
		panic(err)
	}

	defer turnServer.Close()

	// Instantiate the application router
	r, err := handler.New(service, logger, &e.turn)

	if err != nil {
		panic(err)
	}

	defer r.Close()
	var l net.Listener
	if *e.i2p.useI2P {
		l, err = sam.I2PListener("rtcchat", net.JoinHostPort(*e.i2p.samIP, strconv.Itoa(*e.i2p.samPort)), "rtcchat")
		if err != nil {
			panic(err)
		}
		err := generateTLSKeys(l.Addr().(i2pkeys.I2PAddr).Base32(), *e.tls.cert, *e.tls.key)
		if err != nil {
			panic(err)
		}
	} else {
		l, err = net.Listen("tcp", e.web.Address())
		if err != nil {
			panic(err)
		}
		err := generateTLSKeys(l.Addr().String(), *e.tls.cert, *e.tls.key)
		if err != nil {
			panic(err)
		}
	}
	defer l.Close()

	// Launch the HTTP server!

	s := &http.Server{Handler: r.Handler(), Addr: l.Addr().(i2pkeys.I2PAddr).Base32()}

	go func() {

		if err := s.ServeTLS(l, *e.tls.cert, *e.tls.key); err != http.ErrServerClosed {
			panic(err)
		}
	}()

	defer s.Close()

	logger.Info(`HTTP server launched:
	Listening:	%s`, e.web.Address())

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

func (f *turnFlags) Realm() string    { return *f.realm }
func (f *turnFlags) PublicIP() net.IP { return net.ParseIP(*f.publicIP) }
func (f *turnFlags) Port() int        { return *f.port }
func (f *turnFlags) TurnURL() string  { return fmt.Sprintf("turn:%s:%d", *f.publicIP, *f.port) }
func (f *turnFlags) StunURL() string  { return fmt.Sprintf("stun:%s:%d", *f.publicIP, *f.port) }

func (f *webFlags) Address() string { return fmt.Sprintf(":%d", *f.port) }

func (f *turnFlags) SAMAddress() string { return fmt.Sprintf("%s:%d", *f.i2p.samIP, *f.i2p.samPort) }

func generateTLSKeys(host string, certfile string, keyfile string) error {

	if _, err := os.Stat(keyfile); !os.IsNotExist(err) {
		return nil
	}

	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)

	if err != nil {
		return fmt.Errorf("Failed to generate private key: %v", err)
	}
	keyUsage := x509.KeyUsageDigitalSignature

	var notBefore time.Time
	notBefore = time.Now()
	notAfter := notBefore.Add(time.Duration(365 * 24 * 5 * time.Hour))

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	if err != nil {
		return fmt.Errorf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")

	for _, h := range hosts {

		if ip := net.ParseIP(h); ip != nil {

			template.IPAddresses = append(template.IPAddresses, ip)

		} else {

			template.DNSNames = append(template.DNSNames, h)

		}

	}

	/*if *isCA {

		template.IsCA = true

		template.KeyUsage |= x509.KeyUsageCertSign

	}*/

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)

	if err != nil {
		return fmt.Errorf("Failed to create certificate: %v", err)
	}

	certOut, err := os.Create(certfile)

	if err != nil {
		return fmt.Errorf("Failed to open cert.pem for writing: %v", err)
	}

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("Failed to write data to cert.pem: %v", err)
	}

	if err := certOut.Close(); err != nil {
		return fmt.Errorf("Error closing cert.pem: %v", err)
	}

	log.Print("wrote cert.pem\n")

	keyOut, err := os.OpenFile(keyfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return fmt.Errorf("Failed to open key.pem for writing: %v", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)

	if err != nil {
		return fmt.Errorf("Unable to marshal private key: %v", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("Failed to write data to key.pem: %v", err)
	}

	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("Error closing key.pem: %v", err)
	}

	log.Print("wrote key.pem\n")
	return nil
}
