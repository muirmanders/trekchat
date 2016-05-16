// Copyright (c) 2016, RetailNext, Inc.
// This material contains trade secrets and confidential information of
// RetailNext, Inc.  Any use, reproduction, disclosure or dissemination
// is strictly prohibited without the explicit written permission
// of RetailNext, Inc.
// All rights reserved.

package main

import (
	"math/rand"
	"time"
)

type Bot interface {
	Client
	Run()
}

type bot struct {
	server       *server
	name         string
	enhanceCount int
}

func (b *bot) Name() string {
	return b.name
}

func (b *bot) SendCommand(c string, args interface{}) error {
	return nil
}

func (b *bot) Run() {
	for {
		time.Sleep(time.Millisecond * (5000 + time.Duration(rand.Intn(30000))))

		msg := messageArgs{
			Sender: b.name,
		}

		enhanceMessage(b.name, &msg, b.enhanceCount)
		b.enhanceCount++

		b.server.sendMessage(b, msg)
	}
}

func (s *server) initBots() {
	addBot := func(b Bot) {
		s.clients[b.Name()] = b
		s.clientStats[b.Name()] = &clientStats{
			ConnectionCount: 1,
		}

		go b.Run()
	}

	for _, name := range names {
		addBot(&bot{s, name, 0})
	}

	addBot(romulan{s})
}

type romulan struct {
	server *server
}

func (r romulan) Name() string {
	return "not_romulan"
}

func (r romulan) SendCommand(c string, args interface{}) error {
	return nil
}

func (r romulan) Run() {
	for i := 0; true; i++ {
		if i%10000 == 0 {
			r.server.Lock()
			time.Sleep(2 * time.Second)
			r.server.Unlock()
		}
		msg := messageArgs{
			Sender:    r.Name(),
			Private:   true,
			Recipient: r.Name(),
			Message:   "",
		}

		if err := r.server.sendMessage(r, msg); err != nil {
			break
		}
	}
}
