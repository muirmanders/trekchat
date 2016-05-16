package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/retailnext/cannula"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	s := &server{
		clients:     make(map[string]Client),
		clientStats: make(map[string]*clientStats),
	}
	s.initBots()

	cannula.HandleFunc("/debug/chat/status", s.debugStatus)
	cannula.HandleFunc("/debug/chat/user/", s.debugUser)
	cannula.HandleFunc("/debug/chat/private", s.debugPrivate)

	l, err := net.Listen("tcp4", "localhost:8081")
	if err != nil {
		panic(err)
	}
	go cannula.Serve(l)

	http.Handle("/connect", http.HandlerFunc(s.handleConnect))
	http.Handle("/", http.FileServer(http.Dir("static")))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type server struct {
	sync.RWMutex
	clients     map[string]Client
	clientStats map[string]*clientStats
	privateHook func(Client, messageArgs)
}

type clientStats struct {
	BroadcastCount  int64 `json:"broadcast_count"`
	PrivateCount    int64 `json:"private_count"`
	ConnectionCount int64 `json:"connection_count"`
}

type Client interface {
	SendCommand(string, interface{}) error
	Name() string
}

type webClient struct {
	sync.RWMutex
	name string
	conn *websocket.Conn
}

func (c *webClient) Name() string {
	return c.name
}

func (c *webClient) SendCommand(command string, args interface{}) error {
	c.Lock()
	defer c.Unlock()

	err := c.conn.WriteJSON(commandToClient{
		Command: command,
		Args:    args,
	})
	if err != nil {
		log.Printf("Delivery to %s failed: %s", c.name, err)
		return errors.New("delivery failed")
	}
	return nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type commandFromClient struct {
	Command string          `json:"command"`
	Args    json.RawMessage `json:"args"`
}

type messageArgs struct {
	Message   string `json:"message"`
	Private   bool   `json:"private"`
	Recipient string `json:"recipient"`

	// not populated from client
	Sender string `json:"sender"`
	FromMe bool   `json:"from_me"`
}

type commandToClient struct {
	Command string      `json:"command"`
	Args    interface{} `json:"args"`
}

var names = []string{
	"picard",
	"worf",
	"data",
	"barclay",
	"troi",
	"q",
	"crusher",
	"wesley",
	"obrien",
	"laforge",
	"riker",
	"borg",
}

func randomName() string {
	return names[rand.Intn(len(names))]
}

func (s *server) sendMessage(from Client, msg messageArgs) error {
	s.Lock()
	stats := s.clientStats[from.Name()]
	if stats == nil {
		stats = &clientStats{}
		s.clientStats[from.Name()] = stats
	}
	s.Unlock()

	if msg.Private {
		stats.PrivateCount++
		s.RLock()
		recipient := s.clients[msg.Recipient]
		privateHook := s.privateHook
		s.RUnlock()
		if recipient == nil {
			return fmt.Errorf("no such recipient %s", msg.Recipient)
		}
		if privateHook != nil {
			privateHook(from, msg)
		}
		return recipient.SendCommand("message", msg)
	} else {
		stats.BroadcastCount++
		s.broadcastCommand(from, "message", msg)
		return nil
	}
}

func (s *server) broadcastCommand(sender Client, command string, args interface{}) {
	s.RLock()
	defer s.RUnlock()

	for _, c := range s.clients {
		if c == sender {
			continue
		}

		err := c.SendCommand(command, args)

		if err != nil {
			log.Printf("Failed sending message to %s: %s", c.Name(), err)
		}
	}
}

func (s *server) addWebClient(conn *websocket.Conn) *webClient {
	c := &webClient{conn: conn}

	var name string

	s.Lock()
	defer func() {
		if s.clientStats[name] == nil {
			s.clientStats[name] = &clientStats{}
		}
		s.clientStats[name].ConnectionCount++
		s.Unlock()
		s.broadcastUsers()
	}()

	for i := 0; i < 100; i++ {
		name = randomName()
		if s.clients[name] == nil {
			c.name = name
			s.clients[name] = c
			return c
		}
	}

	for {
		name = fmt.Sprintf("cadet#%d", rand.Intn(10000))
		if s.clients[name] == nil {
			c.name = name
			s.clients[name] = c
			return c
		}
	}
}

func (s *server) removeClient(name string) {
	s.Lock()
	delete(s.clients, name)
	s.Unlock()

	s.broadcastUsers()
}

func (s *server) broadcastUsers() {
	var users []string
	s.RLock()
	for _, c := range s.clients {
		users = append(users, c.Name())
	}
	s.RUnlock()
	sort.Strings(users)
	s.broadcastCommand(nil, "users", map[string]interface{}{
		"users": users,
	})
}

var enhancements = map[string][]string{
	"picard": []string{
		"I expect everyone to attend my flute recital tonight.",
		"Tea. Earl Gray. Hot. Oops, wrong terminal.",
		"Beverley, I've pulled something in my groin. Can you come take a look tonight, say 1800 hours?",
	},
	"worf": []string{
		"Captain, ten forward has stopped serving gagh. I recommend we fire all photon torpedoes.",
		"Alexander, you will never be a warrior if you keep playing with those dolls!",
		"I will be AFK for a moment as I apply the second coat of wax to my bat'leth",
		"Why does everyone who enters this chat room suddenly lack honor?",
	},
	"data": []string{
		"Anyone for a game of Strategema?",
		"LOL. I comprehended the humor!",
		"Captain, if we emit a high-intensity gravitron pulse, we might be able to reboot Wesley's computer.",
		"Can anyone watch Spot this weekend? There's an all night Sherlock Holmes marathon on holodeck 2.",
	},
	"barclay": []string{
		"I'll be on the holodeck running a work-related program. I'm not to be disturbed.",
		"I think I may have contracted replicatoritis.",
		"I'm holding a symposium about Barclay's Protomorphosis Syndrome tonight in Ten Forward if anyone is interested.",
	},
	"troi": []string{
		"I'm feeling a strong emotional presence in this chat room.",
		"Barclay, you aren't going to get over your severe spacephobia if you keep skipping our sessions.",
		"My mind is being invaded again for the third time this week!",
	},
	"q": []string{
		"We have an opening for a new q. Any takers other than Wesley?",
		"What is the point to all this? You pitiful humans are pathetic.",
		"I must introduce these emoticons to the continuum :P",
	},
	"crusher": []string{
		"Geordi: can you help me reset the password on my tricorder again?",
		"Wesley, I found the missing hyposprays in your quarters...",
		"Don't be alarmed, but my nano-virus seems to have escaped.",
	},
	"wesley": []string{
		"I think the ship is in grave danger!",
		"Why isn't anyone responding to me? Maybe I need to reload my browser...",
		"Shouldn't it be a bigger deal that space, time and thought are the same thing?",
	},
	"obrien": []string{
		"Doctor, I've broken my arm on the holodeck again!",
		"Keiko, I want a divorce",
		"Who keeps calling for emergency transports for sacks of potatoes? I'm not amused.",
	},
	"laforge": []string{
		"Anyone want to go on a date or something?",
		"Captain, I recommend we flood this chat room with Verteron particles.",
		"A dilithium crystal and an antimatter pod walk into a bar...",
	},
	"riker": []string{
		"I hope to see you all at my master class on how to sit down in a chair.",
		"Data, will you answer that damn phone?!",
		"OOO: on Riza until stardate 41672.9.",
	},
	"borg": []string{
		"Resistance is not futile! lol jk",
		"We are throwing a party over on the cube. All assimilatable life forms are invited! BYOB",
		"You think a chat room frequency modulation can keep us out?",
	},
}

func enhanceMessage(sender string, message *messageArgs, idx int) {
	if l := len(enhancements[sender]); l > 0 {
		message.Message = enhancements[sender][idx%l]
	}
}
