package discovery

import "fmt"

type Server struct {
	Name    string `json:"name"`
	Addr    string `json:"addr"`
	Weight  int    `json:"weight"`
	Version string `json:"version"`
	Ttl     int    `json:"ttl"`
}

func (s Server) buildKey() string {
	if len(s.Version) == 0 {
		return fmt.Sprintf("%s/%s", s.Name, s.Addr)
	}
	return fmt.Sprintf("%s/%s/%s", s.Name, s.Addr, s.Version)
}
