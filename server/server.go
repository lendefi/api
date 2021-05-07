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
	return s.client.GetTotalSupply()
}

func (s *Server) Serve() error {
	http.HandleFunc("/v1/circulating", func(w http.ResponseWriter, req *http.Request) {
		result, err, _ := s.cache.Memoize(req.URL.Path, s.cachedTotalSupply)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			log.Println(err.Error())
			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "text/plain; charset=UTF-8")
		ts := result.(*float64)
		totalSupplyString := strconv.FormatFloat(*ts, 'f', -1, 64)
		_, _ = w.Write([]byte(totalSupplyString))
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
