package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	// resolve consul
	_ "github.com/mbobakov/grpc-consul-resolver"
	"github.com/mbobakov/grpc-consul-resolver/api"
)

var (
	addr string
)

func init() {
	flag.StringVar(&addr, "addr", "consul://localhost:8500/sleeper", "grpc server to connect to")
}

func main() {
	flag.Parse()
	if addr == "" {
		log.Fatalf("addr cannot be empty")
	}

	// fix deprecated method later https://github.com/olivere/grpc/issues/1
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBalancerName("round_robin"), // nolint:staticcheck
	}
	cc, err := grpc.Dial(addr, opts...)
	if err != nil {
		log.Fatalf("Could not connect to server: %v", err)
	}
	grpclog.SetLoggerV2(&writeToLog{logrus.New()})

	sleeperClient := api.NewSleeperClient(cc)
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		d, err := strconv.Atoi(s.Text())
		if err != nil {
			log.Printf("Parsing sleep duration error: %v", err)
			continue
		}
		_, err = sleeperClient.Sleep(context.Background(), &api.SleepDuration{
			Sec: int32(d),
		})
		log.Printf("Slept with error: %v", err)
	}

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
