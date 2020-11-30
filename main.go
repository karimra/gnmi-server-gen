package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var config string
var address string
var tlsCert string
var tlsKey string
var tlsCa string
var interval time.Duration
var rate int
var prometheusAddress string
var numServers int
var port uint16

func main() {
	pflag.StringVarP(&config, "config", "c", "", "config file")
	pflag.StringVarP(&address, "address", "a", "", "server address")
	pflag.Uint16VarP(&port, "port", "p", 57400, "gnmi servers start port")
	pflag.StringVarP(&tlsCert, "tls-cert", "", "", "TLS certificate path")
	pflag.StringVarP(&tlsKey, "tls-key", "", "", "TLS key path")
	pflag.StringVarP(&tlsKey, "tls-ca", "", "", "TLS CA path")
	pflag.DurationVarP(&interval, "interval", "i", time.Second, "sample interval")
	pflag.IntVarP(&rate, "rate", "r", 1, "number of updates per interval")
	pflag.StringVarP(&prometheusAddress, "prometheus-address", "", "", "prometheus server address")
	pflag.IntVarP(&numServers, "num-servers", "", 1, "number of servers")
	pflag.Parse()

	log.SetFlags(log.Ldate | log.Lmicroseconds)

	if prometheusAddress != "" {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		promServer := &http.Server{
			Addr:    prometheusAddress,
			Handler: mux,
		}
		go func() {
			log.Printf("starting prometheus server on %s", prometheusAddress)
			if err := promServer.ListenAndServe(); err != nil {
				log.Printf("prometheus server terminated: %v", err)
			}
		}()
	}
	wg := new(sync.WaitGroup)
	wg.Add(numServers)
	for i := 0; i < numServers; i++ {
		go func(i int) {
			defer wg.Done()
			startServer(i)
		}(i)
	}
	wg.Wait()
}

func startServer(i int) {
	s := new(server)
	var err error
	s.cfg.Interval = interval
	s.cfg.rate = rate
	srvAddress := fmt.Sprintf("%s:%d", address, port+uint16(i))
	s.listener, err = net.Listen("tcp", srvAddress)
	if err != nil {
		log.Printf("listerner failed: %v", err)
		return
	}
	var opts []grpc.ServerOption

	//opts = append(opts, grpc.MaxConcurrentStreams(256))

	if tlsCert != "" && tlsKey != "" {
		tlsConfig := &tls.Config{
			Renegotiation:      tls.RenegotiateNever,
			InsecureSkipVerify: viper.GetBool("skip-verify"),
		}
		err := loadCerts(tlsConfig, tlsCert, tlsKey)
		if err != nil {
			log.Printf("failed loading certificates: %v", err)
		}

		err = loadCACerts(tlsConfig, tlsCa)
		if err != nil {
			log.Printf("failed loading CA certificates: %v", err)
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	//
	if prometheusAddress != "" {
		opts = append(opts, grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor))
	}

	s.grpcServer = grpc.NewServer(opts...)
	gnmi.RegisterGNMIServer(s.grpcServer, s)
	if prometheusAddress != "" {
		grpc_prometheus.Register(s.grpcServer)
	}
	log.Printf("starting gNMI server on %s, interval=%s, rate=%d", srvAddress, interval.String(), rate)
	s.grpcServer.Serve(s.listener)
	defer s.grpcServer.Stop()
}

func loadCerts(tlscfg *tls.Config, tlsCert, tlsKey string) error {
	if tlsCert != "" && tlsKey != "" {
		certificate, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
		if err != nil {
			return err
		}
		tlscfg.Certificates = []tls.Certificate{certificate}
		tlscfg.BuildNameToCertificate()
	}
	return nil
}

func loadCACerts(tlscfg *tls.Config, tlsCa string) error {
	certPool := x509.NewCertPool()
	if tlsCa != "" {
		caFile, err := ioutil.ReadFile(tlsCa)
		if err != nil {
			return err
		}
		if ok := certPool.AppendCertsFromPEM(caFile); !ok {
			return errors.New("failed to append certificate")
		}
		tlscfg.RootCAs = certPool
	}
	return nil
}
