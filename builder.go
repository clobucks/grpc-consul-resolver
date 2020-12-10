package consul

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"
)

// schemeName for the urls
// All target URLs like 'consul://.../...' will be resolved by this resolver.
const schemeName = "consul"

// builder implements resolver.Builder and use for constructing all consul resolvers.
type builder struct{}

func (b *builder) Build(url resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	dsn := strings.Join([]string{schemeName + ":/", url.Authority, url.Endpoint}, "/")
	tgt, err := parseURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid consul URL: %w", err)
	}

	cli, err := api.NewClient(tgt.consulConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the ConsulAPI: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	pipe := make(chan []string)
	go watchConsulService(ctx, cli.Health(), tgt, pipe)
	go populateEndpoints(ctx, cc, pipe)

	return &resolvr{cancelFunc: cancel}, nil
}

// Scheme returns the scheme supported by this resolver.
// Scheme is defined at https://github.com/grpc/grpc/blob/master/doc/naming.md.
func (b *builder) Scheme() string {
	return schemeName
}
