package main

import (
	"fmt"
	"log"
	"net/http"

	"flag"
	"strconv"
)

var key_pem, cert_pem string
var auth_filter string
var listen_port int

func init() {
	flag.StringVar(&key_pem, "key_pem", "/etc/rev-proxy/server.key", "Path to key file")
	flag.StringVar(&cert_pem, "cert_pem", "/etc/rev-proxy/server.crt", "Path to cert file")
	flag.IntVar(&listen_port, "listen_port", 8443, "Port to listen on")
	flag.StringVar(&auth_filter, "auth_filter", "none", "Auth Filter to use. Either 'saml', 'none', or 'db'")
}

var ServiceRegistry = DefaultRegistry{
	"service1": {
		"v1": {
			"192.168.99.100:3000",
		},
	},
	"dhcp": {
		"v1": {
			"192.168.99.100:6755",
		},
	},
	"dns": {
		"v1": {
			"192.168.99.100:6754",
		},
	},
	"provisioner": {
		"v1": {
			"192.168.99.100:8092",
		},
	},
	"default": {
		"default": {
			"192.168.99.100:3000",
		},
	},
}

func main() {
	flag.Parse()

	if auth_filter != "none" {
		// Service multiplexer
		myMux := http.NewServeMux()
		myMux.HandleFunc("/", NewMultipleHostReverseProxy(ServiceRegistry))
		myMux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "%v\n", ServiceRegistry)
		})

		if auth_filter == "saml" {
			NewSamlAuthFilter(myMux, cert_pem, key_pem)
		} else if auth_filter == "db" {
			log.Fatal("Unsupported auth_filter: %v", auth_filter)
		} else {
			log.Fatal("Unknown auth_filter: %v", auth_filter)
		}
	} else {
		http.HandleFunc("/", NewMultipleHostReverseProxy(ServiceRegistry))
		http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "%v\n", ServiceRegistry)
		})
	}

	println("ready")
	log.Fatal(http.ListenAndServeTLS(":"+strconv.Itoa(listen_port), cert_pem, key_pem, nil))
}