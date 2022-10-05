package turn

import (
	"fmt"
	"strconv"

	"github.com/eyedeekay/i2pkeys"
	sam "github.com/eyedeekay/sam3/helper"
	"github.com/yuukanoo/rtchat/internal/logging"
	"github.com/yuukanoo/rtchat/internal/service"

	"github.com/pion/turn/v2"

	"net"
)

type (
	// Options needed by the turn server to be instantiated correclty.
	Options interface {
		// Realm used by the turn server.
		Realm() string
		// PublicIP at which the turn server will be publicly accessible.
		PublicIP() net.IP
		// Port at which the turn server will be made available.
		Port() int
		// SAMAddress at which the Simple Anonymous Messaging bridge can be reached.
		SAMAddress() string
	}

	// Server made available to traverse NAT.
	Server interface {
		// Close the server and stops the listener.
		Close() error
	}
)

// New instantiates a new turn server.
func New(service service.Service, logger logging.Logger, options Options) (Server, error) {
	udpListener, err := sam.I2PDatagramSession("rtcchat-turn", options.SAMAddress(), "rtcchat-turn")

	if err != nil {
		return nil, err
	}

	s, err := turn.NewServer(turn.ServerConfig{
		Realm: options.Realm(),
		AuthHandler: func(username, realm string, srcAddr net.Addr) (key []byte, ok bool) {
			switch srcAddr.(type) {
			case *i2pkeys.I2PAddr:
				// We have an I2P datagram, so it's OK to use
				room := service.GetRoom(username)

				if room == nil {
					return nil, false
				}

				// TODO maybe cache this thing
				return turn.GenerateAuthKey(room.ID, realm, room.Credential), true
			}
			return nil, false
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &I2PRelayAddressGenerator{
					RelayAddress: udpListener.Addr().(i2pkeys.I2PAddr).Base32(), // Claim that we are listening on IP passed by user (This should be your Public IP)
					SAMAddress:   options.SAMAddress(),
				},
			},
		},
	})

	logger.Info(`TURN/STUN Server launched:
	Realm:		%s
	Public IP:	%s
	Port:		%d`, options.Realm(), options.PublicIP(), options.Port())

	return s, err
}

type I2PRelayAddressGenerator struct {
	RelayAddress string
	SAMAddress   string
}

func (i *I2PRelayAddressGenerator) Validate() error {
	switch {
	case i.RelayAddress == "":
		return fmt.Errorf("I2PRelayAddressGenerator: RelayAddress is empty")
	default:
		return nil
	}
}

// Allocate a PacketConn (UDP) RelayAddress
func (i *I2PRelayAddressGenerator) AllocatePacketConn(network string, requestedPort int) (net.PacketConn, net.Addr, error) {
	conn, err := sam.I2PDatagramSession("rtcchat-turn-udp"+strconv.Itoa(requestedPort), i.SAMAddress, "rtcchat-turn-udp"+strconv.Itoa(requestedPort))
	if err != nil {
		return nil, nil, err
	}

	// Replace actual listening IP with the user requested one of RelayAddressGeneratorStatic
	relayAddr, ok := conn.LocalAddr().(*i2pkeys.I2PAddr)
	if !ok {
		return nil, nil, fmt.Errorf("I2PRelayAddressGenerator: AllocatePacketConn: conn.LocalAddr() is not an I2PAddr")
	}

	//relayAddr.IP = i.RelayAddress

	return conn, relayAddr, nil
}

// Allocate a Conn (TCP) RelayAddress
func (i *I2PRelayAddressGenerator) AllocateConn(network string, requestedPort int) (net.Conn, net.Addr, error) {
	/*sess, err := sam.I2PStreamSession("rtcchat-turn-tcp"+strconv.Itoa(requestedPort), i.SAMAddress, "rtcchat-turn-tcp"+strconv.Itoa(requestedPort))
	if err != nil {
		return nil, nil, err
	}
	relayAddr, ok := sess.LocalAddr().(*i2pkeys.I2PAddr)
	if !ok {
		return nil, nil, fmt.Errorf("nil connection error")
	}
	list, err := sess.Listen()
	if err != nil {
		return nil, nil, err
	}
	conn, err := list.Accept()
	if err != nil {
		return nil, nil, err
	}
	return conn, relayAddr, nil*/
	return nil, nil, fmt.Errorf("nil connection error")
}
