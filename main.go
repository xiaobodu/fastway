package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/fast/fastway/proto"
	"github.com/fast/reuseport"
	"github.com/funny/cmd"
	"github.com/funny/slab"
)

var (
	reusePort       = flag.Bool("ReusePort", false, "Enable/Disable the reuseport feature.")
	maxPacketSize   = flag.Int("MaxPacketSize", 512*1024, "Limit max packet size.")
	memPoolType     = flag.String("MemPoolType", "atom", "Type of memory pool ('sync', 'atom' or 'chan').")
	memPoolSize     = flag.Int("MemPoolSize", 10*1024*1024, "Size of memory pool.")
	memPoolFactor   = flag.Int("MemPoolFactor", 2, "Growth in chunk size of memory pool.")
	memPoolMinChunk = flag.Int("MemPoolMinChunk", 64, "Smallest chunk size of memory pool.")
	memPoolMaxChunk = flag.Int("MemPoolMaxChunk", 64*1024, "Largest chunk size of memory pool.")

	clientAddr         = flag.String("ClientAddr", ":0", "The gateway address where clients connect to.")
	clientMaxConn      = flag.Int("ClientMaxConn", 8, "Limit max virtual connections for each client.")
	clientBufferSize   = flag.Int("ClientBufferSize", 2*1024, "Setting bufio.Reader's buffer size.")
	clientSendChanSize = flag.Int("ClientSendChanSize", 1024, "Tunning client session's async behavior.")
	clientPingInterval = flag.Duration("ClientPingInterval", 30*time.Second, "The interval of that gateway sending PING command to client.")

	serverAddr         = flag.String("ServerAddr", ":0", "The gateway address where servers connect to.")
	serverAuthTimeout  = flag.Int("ServerAuthTimeout", 3, "Server auth IO waiting timeout.")
	serverAuthKey      = flag.String("ServerAuthKey", "", "The private key used to auth server connection.")
	serverBufferSize   = flag.Int("ServerBufferSize", 64*1024, "Buffer size of bufio.Reader for server connections.")
	serverSendChanSize = flag.Int("ServerSendChanSize", 102400, "Tunning server session's async behavior, this value must be greater than zero.")
	serverPingInterval = flag.Duration("ServerPingInterval", 30*time.Second, "The interval of that gateway sending PING command to server.")
)

func main() {
	flag.Parse()

	var pool slab.Pool
	switch *memPoolType {
	case "sync":
		pool = slab.NewSyncPool(*memPoolMinChunk, *memPoolMaxChunk, *memPoolFactor)
	case "atom":
		pool = slab.NewAtomPool(*memPoolMinChunk, *memPoolMaxChunk, *memPoolFactor, *memPoolSize)
	case "chan":
		pool = slab.NewChanPool(*memPoolMinChunk, *memPoolMaxChunk, *memPoolFactor, *memPoolSize)
	default:
		println(`unsupported memory pool type, must be "sync" or "chan"`)
	}

	if *serverSendChanSize <= 0 {
		println("server send chan size must greater than zero.")
	}

	gw := proto.NewGateway(pool, *maxPacketSize)

	go gw.ServeClients(listen("client", *clientAddr, *reusePort),
		*clientMaxConn,
		*clientBufferSize,
		*clientSendChanSize,
		*clientPingInterval,
	)

	go gw.ServeServers(listen("server", *serverAddr, *reusePort),
		*serverAuthKey,
		*serverAuthTimeout,
		*serverBufferSize,
		*serverSendChanSize,
		*serverPingInterval,
	)

	cmd.Shell("fastway")

	gw.Stop()
}

func listen(who, addr string, reuse bool) net.Listener {
	var lsn net.Listener
	var err error

	if reuse {
		lsn, err = reuseport.NewReusablePortListener("tcp", addr)
	} else {
		lsn, err = net.Listen("tcp", addr)
	}

	if err != nil {
		log.Fatalf("setup %s listener at %s failed - %s", who, addr, err)
	}

	log.Printf("setup %s listener at - %s", who, lsn.Addr())
	return lsn
}
