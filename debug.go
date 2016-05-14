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
	"sort"
	"strings"
)

func (s *server) debugUsers(w http.ResponseWriter, r *http.Request) {
	var (
		names []string
	)

	s.RLock()
	for _, c := range s.clients {
		names = append(names, c.Name())
	}
	s.RUnlock()

	sort.Strings(names)

	res := make(map[string]*clientStats)

	s.RLock()
	for _, n := range names {
		res[n] = s.clientStats[n]
	}
	s.RUnlock()

	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "error marshaling /users debug output: %s", err)
		return
	}

	w.Write(out)
}

func (s *server) debugUser(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	name := pathParts[len(pathParts)-1]

	s.RLock()
	_, ok := s.clients[name]
	s.RUnlock()
	if !ok {
		w.Write([]byte(fmt.Sprintf("no such user %q\n", name)))
		return
	}

	if r.Method == "DELETE" {
		s.removeClient(name)
		w.Write([]byte("kicked\n"))
	}
}
