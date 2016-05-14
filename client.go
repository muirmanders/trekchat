package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
)

func (s *server) handleConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error making websocket: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sender := s.addWebClient(conn)

	log.Printf("User %s connected", sender.name)

	defer func() {
		log.Printf("User %s disconnected", sender.name)
		s.removeClient(sender.Name())
	}()

	err = sender.SendCommand("welcome", map[string]interface{}{
		"name": sender.name,
	})
	if err != nil {
		log.Printf("Error sending welcome command: %s", err)
		return
	}

	var enhanceCount int

	for {
		var (
			command         commandFromClient
			responseCommand string
			responseArgs    interface{}
			message         messageArgs
		)

		if err := conn.ReadJSON(&command); err != nil {
			log.Printf("error reading command: %s", err)
			return
		}

		switch command.Command {
		case "send_message":
			if err := json.Unmarshal(command.Args, &message); err != nil {
				log.Printf("error unmarshaling message args: %s", err)
				return
			}

			message.Sender = sender.name

			if rand.Intn(2) == 0 {
				enhanceMessage(sender.name, &message, enhanceCount)
				enhanceCount++
			}

			err := s.sendMessage(sender, message)
			if err != nil {
				responseCommand = "error"
				responseArgs = map[string]string{
					"message": err.Error(),
				}
				break
			}

			responseCommand = "message"
			message.FromMe = true
			responseArgs = message
		default:
			log.Printf("unknown command: %s", command.Command)
			return
		}

		if err := sender.SendCommand(responseCommand, responseArgs); err != nil {
			log.Printf("error writing response: %s", err)
			return
		}
	}
}
