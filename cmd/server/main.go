package main

import (
	"flag"

	/*"net"*/

	/*	"net"*/

	"os"
	"os/signal"
	"syscall"

	//sam

	server "github.com/yuukanoo/rtchat/cmd"
	"github.com/yuukanoo/rtchat/internal/logging"
)

func main() {
	e := server.Flags{
		Turn: server.TurnFlags{
			RealmString:    flag.String("realm", "rtchat.io", "Realm used by the turn server."),
			PublicIPString: flag.String("turn-ip", "127.0.0.1", "IP Address that TURN can be contacted on. Should be publicly available."),
			PortInt:        flag.Int("turn-port", 3478, "Listening port for the TURN/STUN endpoint."),
			I2p: server.I2pFlags{
				SamIP:   flag.String("sam-ip", "127.0.0.1", "IP address on which the Simple Anonymous Messaging bridge can be reached"),
				SamPort: flag.Int("sam-port", 7656, "Port on which the Simple Anonymous Messaging bridge can be reached"),
			},
		},
		Web: server.WebFlags{
			Port: flag.Int("http-port", 5000, "Web server listening port."),
		},
	}

	flag.Parse()
	addr := server.Serve(e, "rtchat")
	defer server.Close()
	logger := logging.New(false)
	logger.Info(addr)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	logger.Info("Shutting down, goodbye 👋")
}
