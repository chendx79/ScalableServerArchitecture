package service

import (
	"bytes"
	"errors"
	"net"
	"time"

	"code.google.com/p/go-uuid/uuid"
	json "github.com/bitly/go-simplejson"
	zmq "github.com/pebbe/zmq4"
)

// =====================================================================
// Synchronous part, works in our application thread

// ---------------------------------------------------------------------
// Structure of our class

type Callback struct {
	found func(ip string)
	lost  func(ip string)
}

type Service struct {
	servType   string               // Type of service
	servFilter map[string]*Callback // Type of service interested in
}

// This is the thread that handles our real interface class

// Here is the constructor for the interface class.
// Note that the class has barely any properties, it is just an excuse
// to start the background thread, and a wrapper around zmsg_recv():

func New(servType string) (srv *Service) {
	return &Service{
		servType:   servType,
		servFilter: make(map[string]*Callback),
	}
}

func (srv *Service) Discover(servType string, found func(string), lost func(string)) {
	srv.servFilter[servType] = &Callback{found, lost}
}

func (srv *Service) Start() {
	go srv.agent()
}

// Temporary beacon message definition for demonstration purpose.
type Beacon struct {
	Type   string
	UUID   string
	IpAddr string
}

func Serialize(buffer []byte, addr string) (beacon *Beacon, err error) {
	jsonMsg, _ := json.NewJson(buffer)
	beacon = &Beacon{}
	beacon.Type = jsonMsg.Get("type").MustString()
	beacon.UUID = jsonMsg.Get("uuid").MustString()
	beacon.IpAddr = addr
	return
}

func Deserialize(msg []string) (beacon *Beacon, err error) {
	if len(msg) != 3 && len(msg[1]) != 16 {
		return nil, errors.New("Not a valid message")
	}

	beacon = &Beacon{}
	beacon.Type = msg[0]
	beacon.UUID = msg[1]
	beacon.IpAddr = msg[2]
	return
}

//=====================================================================
// Asynchronous part, works in the background
//
// This structure defines each peer that we discover and track:

type peer_t struct {
	peer_type     string
	uuid_bytes    []byte
	uuid_string   string
	ipaddr_bytes  []byte
	ipaddr_string string
	expires_at    time.Time
}

// We have a constructor for the peer class:

// func new_peer(peer_type string, uuid uuid.UUID, ipaddr []byte) (peer *peer_t) {
func new_peer(beacon *Beacon) (peer *peer_t) {
	peer = &peer_t{
		peer_type:     beacon.Type,
		uuid_bytes:    uuid.Parse(beacon.UUID),
		uuid_string:   beacon.UUID,
		ipaddr_bytes:  []byte(net.ParseIP(beacon.IpAddr)),
		ipaddr_string: beacon.IpAddr,
	}
	return
}

// Just resets the peers expiry time; we call this method
// whenever we get any activity from a peer.

func (peer *peer_t) is_alive() {
	peer.expires_at = time.Now().Add(PEER_EXPIRY)
}

// This structure holds the context for our agent, so we can
// pass that around cleanly to methods which need it:

type agent_t struct {
	udp         *zmq.Socket
	conn        *net.UDPConn
	servType    string
	servFilter  map[string]*Callback
	uuid_bytes  []byte // Our UUID
	uuid_string string
	servers     map[string]map[string]*peer_t // Known servers, classified by service type
}

// Now the constructor for our agent. Each interface
// has one agent object, which implements its background thread:

func new_agent(servType string, servFilter map[string]*Callback) (agent *agent_t) {
	// push output from udp into zmq socket
	bcast := &net.UDPAddr{Port: PING_PORT_NUMBER, IP: net.IPv4bcast}
	conn, e := net.ListenUDP("udp", bcast)
	if e != nil {
		panic(e)
	}
	go func() {
		buffer := make([]byte, 1024)
		udp, _ := zmq.NewSocket(zmq.PAIR)
		udp.Bind("inproc://udp")

		for {
			if n, addr, err := conn.ReadFrom(buffer); err == nil {
				addr_string, _, _ := net.SplitHostPort(addr.String())
				beacon, _ := Serialize(buffer[:n], addr_string)
				udp.SendMessage(beacon.Type, beacon.UUID, beacon.IpAddr)
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)

	udp, _ := zmq.NewSocket(zmq.PAIR)
	udp.Connect("inproc://udp")

	uuid := uuid.NewRandom()
	agent = &agent_t{
		udp:         udp,
		conn:        conn,
		servType:    servType,
		servFilter:  servFilter,
		uuid_bytes:  []byte(uuid),
		uuid_string: uuid.String(),
		servers:     make(map[string]map[string]*peer_t),
	}

	return
}

// This is how we handle a beacon coming into our UDP socket;
// this may be from other peers or an echo of our own broadcast
// beacon:

func (agent *agent_t) handle_beacon() (err error) {

	msg, err := agent.udp.RecvMessage(0)

	// If we got a UUID and it's not our own beacon, we have a peer
	beacon, _ := Deserialize(msg)

	// See if the broadcasting server is our interest
	callback, interested := agent.servFilter[beacon.Type]
	if !interested {
		return
	}

	if callback == nil || callback.found == nil {
		panic("Callback not registered")
	}

	// We are interested in this kind of service, see if it's already registered.
	uuid_bytes := uuid.Parse(beacon.UUID)
	if bytes.Compare(uuid_bytes, agent.uuid_bytes) != 0 {
		// Find or create peer via its UUID string
		serverMap, exist := agent.servers[beacon.Type]

		// If there is no server of this type yet, create it.
		if !exist {
			serverMap = make(map[string]*peer_t)
		}

		// Find peer in registered server, register it if not yet.
		peer, ok := serverMap[beacon.UUID]
		if !ok {
			peer = new_peer(beacon)
			serverMap[beacon.UUID] = peer

			// Report peer joined the network
			callback.found(beacon.IpAddr)
		}

		if !exist {
			// Update server map
			agent.servers[beacon.Type] = serverMap
		}
		// Any activity from the peer means it's alive
		peer.is_alive()
	}
	return
}

// This method checks one peer item for expiry; if the peer hasn't
// sent us anything by now, it's 'dead' and we can delete it:

func (agent *agent_t) reap_peer(peer *peer_t) {
	if time.Now().After(peer.expires_at) {
		// Report peer left the network
		callback, ok := agent.servFilter[peer.peer_type]
		if !ok {
			panic("Server type not found in server filter")
		}
		if callback == nil || callback.lost == nil {
			panic("No callback registered")
		}
		callback.lost(peer.ipaddr_string)

		delete(agent.servers[peer.peer_type], peer.uuid_string)
	}
}

// This is the main loop for the background agent. It uses zmq_poll
// to monitor the front-end pipe (commands from the API) and the
// back-end UDP handle (beacons):

func (srv *Service) agent() {
	// Create agent instance to pass around
	agent := new_agent(srv.servType, srv.servFilter)

	// Send first beacon immediately
	ping_at := time.Now()

	poller := zmq.NewPoller()
	poller.Add(agent.udp, zmq.POLLIN)

	bcastMsg := json.New()
	bcast := &net.UDPAddr{Port: PING_PORT_NUMBER, IP: net.IPv4bcast}
	for {
		timeout := ping_at.Add(time.Millisecond).Sub(time.Now())
		if timeout < 0 {
			timeout = 0
		}
		polled, err := poller.Poll(timeout)
		if err != nil {
			break
		}

		for _, item := range polled {
			switch socket := item.Socket; socket {
			case agent.udp:
				// If we had input on the UDP socket, go process that
				agent.handle_beacon()
			}
		}

		// If we passed the 1-second mark, broadcast our beacon
		now := time.Now()
		if now.After(ping_at) {
			bcastMsg.Set("type", agent.servType)
			bcastMsg.Set("uuid", agent.uuid_string)
			bcastStr, _ := bcastMsg.MarshalJSON()
			agent.conn.WriteTo(bcastStr, bcast)
			ping_at = now.Add(PING_INTERVAL)
		}
		// For each kind of server, report and delete expired peers.
		for _, server := range agent.servers {
			for _, peer := range server {
				agent.reap_peer(peer)
			}
		}
	}
}
