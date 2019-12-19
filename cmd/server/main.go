package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hashicorp/go-uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	cApi "github.com/hashicorp/consul/api"

	"github.com/mbobakov/grpc-consul-resolver/api"
)

var (
	port        int
	gracePeriod int
)

func init() {
	flag.IntVar(&port, "port", 0, "grpc port to serve on")
	flag.IntVar(&gracePeriod, "grace", 60, "graceful shutdown period")
}

func main() {
	flag.Parse()

	if port == 0 {
		log.Fatal("port should not be empty")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("couldn't start listen on the %d: %v", port, err)
	}

	consulClient, err := cApi.NewClient(&cApi.Config{
		Address: "http://localhost:8500",
		TLSConfig: cApi.TLSConfig{
			InsecureSkipVerify: true,
		},
	})
	if err != nil {
		log.Fatalf("couldn't connect to the Consul API: %v", err)
	}

	id, err := uuid.GenerateUUID()
	if err != nil {
		log.Fatalf("Could not gen id: %v", err)
	}
	log.Printf("Registerig oneself as %s", id)

	err = consulClient.Agent().ServiceRegister(&cApi.AgentServiceRegistration{
		ID:      id,
		Name:    "sleeper",
		Port:    port,
		Address: "127.0.0.1",
		Check: &cApi.AgentServiceCheck{
			CheckID:       id,
			Name:          "grpc healthz",
			Interval:      "5s",
			TLSSkipVerify: false,
			GRPC:          fmt.Sprintf("127.0.0.1:%d", port),
			GRPCUseTLS:    false,
		},
	})
	if err != nil {
		log.Fatalf("Couldn't register in Consul: %v", err)
	}

	srv := grpc.NewServer()
	api.RegisterSleeperServer(srv, sleeper{id: id})
	grpc_health_v1.RegisterHealthServer(srv, health.NewServer())

	srvErr := make(chan error, 1)
	go func() {
		log.Printf("Serving on %d", port)
		srvErr <- srv.Serve(lis)
		close(srvErr)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	s := <-sigs
	log.Printf("Caught %s signal. Stopping the server...", s)
	srv.GracefulStop()

	select {
	case err := <-srvErr:
		log.Printf("Server gracefully finished with error: %v", err)
	case <-time.After(time.Second * time.Duration(gracePeriod)):
		srv.Stop()
		log.Printf("Server forcibly shut down with error: %v", <-srvErr)
	}

	log.Printf("Deregisterig oneself as %s", id)
	err = consulClient.Agent().ServiceDeregister(id)
	if err != nil {
		log.Fatalf("Couldn't deregister in Consul: %v", err)
	}
}

type sleeper struct {
	id string
}

func (s sleeper) Sleep(_ context.Context, d *api.SleepDuration) (*empty.Empty, error) {
	log.Printf("[%s] Sleeping for %d seconds", s.id, d.Sec)
	time.Sleep(time.Second * time.Duration(d.Sec))
	return &empty.Empty{}, nil
}

type writeToLog struct {
	logrus.FieldLogger
}

func (w *writeToLog) V(lvl int) bool {
	return true
}

// nolint: unparam
func (w *writeToLog) Write(p []byte) (int, error) {
	w.FieldLogger.Errorf("%s", p)
	return len(p) + 1, nil
}
