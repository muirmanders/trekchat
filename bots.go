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

func (b *bot) run() {
	for {
		time.Sleep(time.Second * (10 + time.Duration(rand.Intn(5))))

		msg := messageArgs{
			Sender: b.name,
		}

		enhanceMessage(b.name, &msg, b.enhanceCount)
		b.enhanceCount++

		b.server.broadcastCommand(b, "message", msg)
	}
}

func (s *server) initBots() {
	for i := 0; i < 5; i++ {
		name := randomName()
		if s.clients[name] != nil {
			i--
			continue
		}

		b := &bot{s, name, 0}
		s.clients[name] = b
		go b.run()
	}
}
