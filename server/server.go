package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kofalt/go-memoize"
	"github.com/lendefi/api/ldfi"
)

type Server struct {
	client  *ldfi.Client
	cache   *memoize.Memoizer
	address string
}

func (s *Server) cachedTotalSupply() (interface{}, error) {
	return s.client.GetSupplies()
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	_, wErr := w.Write([]byte(err.Error()))
	if wErr != nil {
		log.Printf("Can't write error %q: %q\n", err.Error(), wErr.Error())
		return
	}
	log.Printf("Error: %q\n", err.Error())
}

func writeFloat(w http.ResponseWriter, f float64) {
	w.WriteHeader(http.StatusAccepted)
	w.Header().Add("Content-Type", "text/plain; charset=UTF-8")
	str := strconv.FormatFloat(f, 'f', -1, 64)
	_, err := w.Write([]byte(str))
	if err != nil {
		log.Printf("Can't write response: %q\n", err.Error())
	}
}

func (s *Server) Serve() error {
	http.HandleFunc("/v2/circulating", func(w http.ResponseWriter, req *http.Request) {
		result, err, _ := s.cache.Memoize(req.URL.Path, s.cachedTotalSupply)
		if err != nil {
			writeError(w, err)
			return
		}

		tr := result.(*ldfi.Supplies)
		writeFloat(w, tr.Circulating)
	})

	http.HandleFunc("/v2/total", func(w http.ResponseWriter, req *http.Request) {
		result, err, _ := s.cache.Memoize(req.URL.Path, s.cachedTotalSupply)
		if err != nil {
			writeError(w, err)
			return
		}

		tr := result.(*ldfi.Supplies)
		writeFloat(w, tr.Total)
	})

	http.HandleFunc("/v2/max", func(w http.ResponseWriter, req *http.Request) {
		result, err, _ := s.cache.Memoize(req.URL.Path, s.cachedTotalSupply)
		if err != nil {
			writeError(w, err)
			return
		}

		tr := result.(*ldfi.Supplies)
		writeFloat(w, tr.Max)
	})

	log.Println("Listen to http server:", s.address)
	err := http.ListenAndServe(s.address, nil)
	if err != nil {
		return fmt.Errorf("Can't listen to %q: %w", s.address, err)
	}

	return nil
}

func NewServer(client *ldfi.Client, address string, cacheExpiration time.Duration) *Server {
	return &Server{
		address: address,
		client:  client,
		// cleanup interval is not really import as there will be only 1 key
		cache: memoize.NewMemoizer(cacheExpiration, time.Hour),
	}
}

// NewServerFromEnv is like NewServer but initialize itself through environment variables.
// LISTEN_ADDRESS: default ':8080'
// CACHE_TIMEOUT default '1m'
func NewServerFromEnv(client *ldfi.Client) (*Server, error) {
	const (
		envKeyAddress                  = "LISTEN_ADDRESS"
		envKeyTimeout                  = "CACHE_TIMEOUT"
		errFormatMissingEnvironmentKey = "Missing environment key %q"
	)

	var (
		address         = os.Getenv(envKeyAddress)
		cacheTimeoutStr = os.Getenv(envKeyTimeout)
	)

	if len(cacheTimeoutStr) == 0 {
		cacheTimeoutStr = "1m"
	}

	if len(address) == 0 {
		address = ":8080"
	}

	cacheTimeout, err := time.ParseDuration(cacheTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid duration %q: %w", cacheTimeoutStr, err)
	}

	return NewServer(client, address, cacheTimeout), nil
}
