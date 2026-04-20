package metrics

import (
	"net/http"

	"github.com/arl/statsviz"
)

func Serve(addr string) error {
	mux := http.NewServeMux()
	if err := statsviz.Register(mux); err != nil {
		return err
	}
	if err := http.ListenAndServe(addr, mux); err != nil {
		return err
	}
	return nil
}
