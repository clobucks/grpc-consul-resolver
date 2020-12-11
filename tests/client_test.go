//+build integration

package tests

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	_ "github.com/mbobakov/grpc-consul-resolver"
)

func TestClient(t *testing.T) {
	logger := logrus.New()
	grpclog.SetLoggerV2(&grpcLog{logger})
	conn, err := grpc.Dial("consul://127.0.0.1:8500/whoami?wait=14s&tag=public", grpc.WithInsecure(), grpc.WithBalancerName("round_robin"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	time.Sleep(29 * time.Second)

}

type grpcLog struct {
	*logrus.Logger
}

func (l *grpcLog) V(level int) bool {
	return l.IsLevelEnabled(logrus.Level(level))
}
