package cmd

import (
	"context"
	"io"
	"net/url"
	"os"

	"github.com/go-logr/logr"
	"github.com/netbox-community/go-netbox/netbox/client"
	"github.com/networkop/declarative-netbox/netbox"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Cli struct {
	netbox *netbox.NetboxServer
	ctx    context.Context
	Out    io.Writer
	Err    io.Writer
}

type Resource struct {
	Name   string
	Apply  func() *cobra.Command
	Delete func() *cobra.Command
	Get    func() *cobra.Command
}

type CliOption func(cli *Cli) error

func NewCli(opts ...CliOption) (*Cli, error) {
	cli := Cli{
		Out: os.Stdout,
		Err: os.Stderr,
		ctx: context.Background(),
	}

	if err := cli.Apply(opts...); err != nil {
		return nil, err
	}

	auth := NewAuthData()

	url, err := url.Parse(auth.Server)
	if err != nil {
		return nil, err
	}

	client.DefaultSchemes = []string{url.Scheme}
	cli.netbox = netbox.NewNetboxServer(url.Host, auth.Token)

	return &cli, nil
}

func (c *Cli) Apply(opts ...CliOption) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return err
		}
	}
	return nil
}

func WithLogger(l logr.Logger) CliOption {
	return func(c *Cli) error {
		c.ctx = log.IntoContext(c.ctx, l)
		return nil
	}
}
