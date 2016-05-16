// Copyright (c) 2016, RetailNext, Inc.
// This material contains trade secrets and confidential information of
// RetailNext, Inc.  Any use, reproduction, disclosure or dissemination
// is strictly prohibited without the explicit written permission
// of RetailNext, Inc.
// All rights reserved.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *server) debugStatus(w http.ResponseWriter, r *http.Request) {
	var (
		users, msgs, private int64
	)
	s.RLock()
	for _, s := range s.clientStats {
		msgs += s.BroadcastCount
		private += s.PrivateCount
		users++
	}
	s.RUnlock()

	fmt.Fprintf(w, "Uers: %d\nMessages: %d\nPrivate Messages: %d\n", users, msgs, private)
}

func (s *server) debugUser(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	name := pathParts[len(pathParts)-1]

	s.RLock()
	_, ok := s.clients[name]
	stats := s.clientStats[name]
	s.RUnlock()
	if !ok {
		fmt.Fprintf(w, "no such user %q\n", name)
		return
	}

	if r.Method == "GET" {
		encoder := json.NewEncoder(w)
		encoder.Encode(stats)
	} else if r.Method == "DELETE" {
		s.removeClient(name)
		fmt.Fprintf(w, "%q black-listed\n", name)
	}
}

func (s *server) debugPrivate(w http.ResponseWriter, r *http.Request) {
	done := make(chan struct{})

	s.Lock()
	s.privateHook = func(from Client, msg messageArgs) {
		_, err := fmt.Fprintf(w, "%s private message from %q to %q\n", time.Now().UTC().Format(time.RFC3339), from.Name(), msg.Recipient)
		if err != nil {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	}
	s.Unlock()

	defer func() {
		s.Lock()
		s.privateHook = nil
		s.Unlock()
	}()

	<-done
}
