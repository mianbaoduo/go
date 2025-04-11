package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/kellegous/go/internal/backend"
	"github.com/kellegous/go/internal/backend/redis"
	"github.com/kellegous/go/internal/web"
)

func getBackend() (backend.Backend, error) {
	switch viper.GetString("backend") {
	case "redis":
		return redis.New(
			viper.GetString("redis-addr"),
			viper.GetString("redis-password"),
			viper.GetInt("redis-db"),
			viper.GetString("redis-prefix"),
		)
	default:
		return nil, fmt.Errorf("unknown backend %s", viper.GetString("backend"))
	}
}

type URL struct {
	*url.URL
}

func (u *URL) Set(v string) error {
	var err error
	u.URL, err = url.Parse(v)
	return err
}

func (u *URL) Type() string {
	return "url"
}

func (u *URL) String() string {
	if u.URL == nil {
		return ""
	}
	return u.URL.String()
}

func main() {
	var assetProxyURL URL
	pflag.String("addr", ":8067", "default bind address")
	pflag.Bool("admin", false, "allow admin-level requests")
	pflag.String("version", "", "version string")
	pflag.String("backend", "redis", "backing store to use. 'redis' currently supported.")
	pflag.String("host", "", "The host field to use when gnerating the source URL of a link. Defaults to the Host header of the generate request")
	pflag.String("redis-addr", "localhost:6379", "Redis server address")
	pflag.String("redis-password", "", "Redis password")
	pflag.Int("redis-db", 0, "Redis database number")
	pflag.String("redis-prefix", "go:routes", "Redis key prefix")
	pflag.String("api-key", "", "API key for internal access")
	pflag.Var(
		&assetProxyURL,
		"asset-proxy-url",
		"The URL to use for proxying asset requests")
	pflag.Parse()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		log.Panic(err)
	}

	// allow env vars to set pflags
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	backend, err := getBackend()
	if err != nil {
		log.Panic(err)
	}
	defer backend.Close()

	log.Panic(web.ListenAndServe(backend))
}
