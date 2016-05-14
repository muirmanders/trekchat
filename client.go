package main

import (
	"encoding/json"
	"fmt"
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
		s.removeClient(sender)
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

			if message.Private {
				s.RLock()
				recipient := s.clients[message.Recipient]
				s.RUnlock()

				if recipient == nil {
					responseCommand = "error"
					responseArgs = map[string]string{
						"error":   "no_such_recipient",
						"message": fmt.Sprintf("%q is not in the chat room.", message.Recipient),
					}
					break
				}

				err := recipient.SendCommand("message", message)

				if err != nil {
					log.Printf("Failed sending message to %s: %s", recipient.Name(), err)
					responseCommand = "error"
					responseArgs = map[string]string{
						"error":   "delivery_failed",
						"message": fmt.Sprintf("Your message to %s could not be delivered.", message.Recipient),
					}
					break
				}

				responseCommand = "message"
				message.FromMe = true
				responseArgs = message
			} else {
				s.broadcastCommand(sender, "message", message)

				responseCommand = "message"
				message.FromMe = true
				responseArgs = message
			}
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
