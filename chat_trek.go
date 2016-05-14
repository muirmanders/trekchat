package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	s := &server{
		clients: make(map[string]*client),
	}
	http.Handle("/connect", http.HandlerFunc(s.handleConnect))
	http.Handle("/", http.FileServer(http.Dir("static")))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type server struct {
	sync.RWMutex
	clients map[string]*client
}

type client struct {
	sync.RWMutex
	name string
	conn *websocket.Conn
}

func (c *client) sendCommand(command string, args interface{}) error {
	c.Lock()
	defer c.Unlock()

	return c.conn.WriteJSON(commandToClient{
		Command: command,
		Args:    args,
	})
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
	"lwaxana",
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

func (s *server) broadcastCommand(sender *client, command string, args interface{}) {
	s.RLock()
	defer s.RUnlock()

	for _, c := range s.clients {
		if c == sender {
			continue
		}

		err := c.sendCommand(command, args)

		if err != nil {
			log.Printf("Failed sending message to %s: %s", c.name, err)
		}
	}
}

func (s *server) addClient(conn *websocket.Conn) *client {
	c := &client{conn: conn}

	s.Lock()
	defer func() {
		s.Unlock()
		s.broadcastUsers()
	}()

	for i := 0; i < 100; i++ {
		name := randomName()
		if s.clients[name] == nil {
			c.name = name
			s.clients[name] = c
			return c
		}
	}

	for {
		name := fmt.Sprintf("cadet#%d", rand.Intn(10000))
		if s.clients[name] == nil {
			c.name = name
			s.clients[name] = c
			return c
		}
	}
}

func (s *server) removeClient(c *client) {
	s.Lock()
	delete(s.clients, c.name)
	s.Unlock()

	s.broadcastUsers()
}

func (s *server) broadcastUsers() {
	var users []string
	s.RLock()
	for _, c := range s.clients {
		users = append(users, c.name)
	}
	s.RUnlock()
	sort.Strings(users)
	s.broadcastCommand(nil, "users", map[string]interface{}{
		"users": users,
	})
}

var enhancements = map[string][]string{
	"picard": []string{
		"Tea. Earl Gray. Hot.",
		"Broccoli, is that report ready yet?",
		"STFU Wesley.",
	},
	"worf": []string{
		"Captain, I recommend we fire all photon torpedos.",
		"Alexander, you will never be a warrior if you keep playing with those dolls!",
		"AFK waxing my bat'leth",
	},
	"data": []string{
		"LOL. I totally understand the humor!",
		"Anyone for a game of Strategema?",
		"Captain, if we emit a high-intensity gravitron pulse, we might be able to reboot Wesley's computer.",
		"Can anyone watch Spot this weekend?",
	},
	"barclay": []string{
		"I'll be on the holodeck. I'm not to be disturbed.",
		"I'm holding a symposium about Barclay's Protomorphosis Syndrome tonight in Ten Forward if anyone is interested.",
	},
	"troi": []string{
		"I'm feeling a strong emotional presence in this chat room.",
		"My mind is being invaded again for the third time this week!",
	},
	"lwaxana": []string{
		"Captain, you should be ashamed for thinking that.",
		"Diana, I took the liberty of arranging a romantic encounter for you and Will ;)",
	},
	"q": []string{
		"What is the point to all this?",
		"You pitiful humans are pathetic.",
		"I must introduce these emojis to the continuum :P",
	},
	"crusher": []string{
		"Has anyone seen my cortical stimulator?",
		"Don't be alarmed, but my nano-virus seems to have escaped.",
	},
	"wesley": []string{
		"I think the ship is in grave danger!",
		"Why isn't anyone responding to me? Maybe I need to reload my browser...",
	},
	"obrien": []string{
		"Doctor, I've broken my arm on the holodeck again!",
		"Keiko, I want a divorce",
	},
	"laforge": []string{
		"Anyone want to go on a date or something?",
		"Captain, I recommend we flood Cargo Bay 2 with Verteron particles.",
	},
	"riker": []string{
		"Data, will you answer that damn phone?!",
		"OOO: on Riza until stardate 41672.9.",
	},
	"borg": []string{
		"Resistance is not futile! lol jk",
		"We are throwing a party over on the cube. All assimilatable life forms are invited! BYOB",
	},
}

func enhanceMessage(sender string, message *messageArgs) {
	if l := len(enhancements[sender]); l > 0 && rand.Intn(2) == 0 {
		message.Message = enhancements[sender][rand.Intn(l)]
	}
}
