package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/itbellissimo/ratelimit/handler"
	"github.com/itbellissimo/ratelimit/middleware"
	"github.com/itbellissimo/ratelimit/pkg/ratelimit"
	"github.com/itbellissimo/ratelimit/pkg/ratelimit/storage"
	"github.com/spf13/viper"
	"log"
	"net/http"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config/")

	cfg, err := getConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	memStorage := storage.NewMemoryCache()
	rateLimit := ratelimit.NewRateLimit(&cfg, memStorage)

	viper.OnConfigChange(func(e fsnotify.Event) {
		cfg, err = getConfig()
		if err != nil {
			log.Fatal(err.Error())
		}
	})
	viper.WatchConfig()

	mux := http.NewServeMux()
	server := handler.NewHandler()
	mux.HandleFunc("/run", server.Run)
	mux.HandleFunc("/reset", server.Reset)

	h := middleware.RateLimit(mux, rateLimit)

	port := "3000"
	portConfig := viper.Get("server.port")
	if portConfig != nil {
		port = fmt.Sprintf("%v", portConfig)
	}

	err = http.ListenAndServe(":"+port, h)
	if err != nil {
		panic(err.Error())
	}
	log.Println("Start HTTP server: 127.0.0.1:" + port)
}

func getConfig() (ratelimit.Config, error) {
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			return ratelimit.Config{}, fmt.Errorf("config file not found. Fatal error read config file: %w", err)
		} else {
			// Config file was found but another error was produced
			return ratelimit.Config{}, fmt.Errorf("read config file fatal error: %w", err)
		}
	}

	var rawVal ratelimit.Config
	err := viper.UnmarshalKey("server.rate_limits", &rawVal)
	if err != nil {
		return ratelimit.Config{}, fmt.Errorf("fatal error config file: %w", err)
	}

	return rawVal, nil
}
