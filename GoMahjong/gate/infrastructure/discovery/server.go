package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Server struct {
	Domain  string  `json:"domain"`
	Addr    string  `json:"addr"`
	Weight  int     `json:"weight"`
	Version string  `json:"version"`
	Ttl     int     `json:"ttl"`
	Load    float64 `json:"load"`
	NodeID  string  `json:"nodeID"`
}

func (s Server) buildKey() string {
	if len(s.Version) == 0 {
		return fmt.Sprintf("%s/%s", s.Domain, s.Addr)
	}
	return fmt.Sprintf("%s/%s/%s", s.Domain, s.Version, s.Addr)
}

func ParseValue(v []byte) (Server, error) {
	var server Server
	if err := json.Unmarshal(v, &server); err != nil {
		return server, err
	}
	return server, nil
}

func ParseKey(key string) (Server, error) {
	strs := strings.Split(key, "/")
	if len(strs) == 2 {
		return Server{
			Domain: strs[0],
			Addr:   strs[1],
		}, nil
	}
	if len(strs) == 3 {
		return Server{
			Domain:  strs[0],
			Addr:    strs[2],
			Version: strs[1],
		}, nil
	}
	return Server{}, errors.New("invalid key")
}
