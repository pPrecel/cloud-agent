package state

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	command "github.com/pPrecel/cloud-agent/cmd"
	"github.com/pPrecel/cloud-agent/internal/output"
	"github.com/pPrecel/cloud-agent/pkg/agent"
	cloud_agent "github.com/pPrecel/cloud-agent/pkg/agent/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	googlerpc "google.golang.org/grpc"
)

func TestNewCmd(t *testing.T) {
	o := NewOptions(&command.Options{})
	c := NewCmd(o)

	t.Run("defaults", func(t *testing.T) {
		assert.Equal(t, "", o.createdBy)
		assert.Equal(t, *output.New(&output.Output{}, "table", "%r/%h/%u/%a", "-/-/-/-"), o.outFormat)
		assert.Equal(t, 2*time.Second, o.timeout)
	})

	t.Run("parse flags", func(t *testing.T) {
		c.ParseFlags([]string{
			"--createdBy", "owner",
			"--output", "text=%a=%e",
			"--timeout", "5s",
		})

		assert.Equal(t, "owner", o.createdBy)
		assert.Equal(t, *output.New(&output.Output{}, "text", "%a", "%e"), o.outFormat)
		assert.Equal(t, 5*time.Second, o.timeout)
	})

	t.Run("parse shortcuts", func(t *testing.T) {
		c.ParseFlags([]string{
			"-c", "other-owner",
			"-o", "text=%a%a%a=%e%e%e",
			"-t", "10s",
		})

		assert.Equal(t, "other-owner", o.createdBy)
		assert.Equal(t, *output.New(&output.Output{}, "text", "%a%a%a", "%e%e%e"), o.outFormat)
		assert.Equal(t, 10*time.Second, o.timeout)
	})
}

var (
	socketAddress = filepath.Join(os.TempDir(), "state-test-socket.sock")
	socketNetwork = "unix"
)

func Test_run(t *testing.T) {
	l := logrus.New()
	l.Out = ioutil.Discard

	t.Run("run and print text", func(t *testing.T) {
		c := agent.NewCache[*v1beta1.ShootList]()
		r := c.Register("test-data")
		stopFn, err := fixServer(l, c)
		assert.NoError(t, err)
		defer stopFn()

		o := &options{
			socketAddress: socketAddress,
			socketNetwork: socketNetwork,
			writer:        io.Discard,
			Options: &command.Options{
				Logger:  l,
				Context: context.Background(),
			},
		}
		cmd := NewCmd(o)
		o.createdBy = "owner"
		o.outFormat = *output.New(&o.outFormat, output.TextType, "%r/%h/%u/%a", "-/-/-/-")

		r.Set(&v1beta1.ShootList{
			Items: []v1beta1.Shoot{
				{}, {}, {},
			},
		})

		err = cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("run and print text error", func(t *testing.T) {
		c := agent.NewCache[*v1beta1.ShootList]()
		r := c.Register("test-data")
		stopFn, err := fixServer(l, c)
		assert.NoError(t, err)
		defer stopFn()

		o := &options{
			socketAddress: socketAddress,
			socketNetwork: socketNetwork,
			writer:        io.Discard,
			Options: &command.Options{
				Logger:  l,
				Context: context.Background(),
			},
		}
		cmd := NewCmd(o)
		o.createdBy = "owner"
		o.outFormat = *output.New(&o.outFormat, output.TextType, "", "-/-/-/-")

		r.Set(&v1beta1.ShootList{
			Items: []v1beta1.Shoot{},
		})

		err = cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("run and print json", func(t *testing.T) {
		c := agent.NewCache[*v1beta1.ShootList]()
		r := c.Register("test-data")
		stopFn, err := fixServer(l, c)
		assert.NoError(t, err)
		defer stopFn()

		o := &options{
			socketAddress: socketAddress,
			socketNetwork: socketNetwork,
			writer:        io.Discard,
			Options: &command.Options{
				Logger:  l,
				Context: context.Background(),
			},
		}
		cmd := NewCmd(o)
		o.createdBy = "owner"
		o.outFormat = *output.New(&o.outFormat, output.JsonType, "", "")

		r.Set(&v1beta1.ShootList{
			Items: []v1beta1.Shoot{
				{}, {}, {},
			},
		})

		err = cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("run and print table", func(t *testing.T) {
		c := agent.NewCache[*v1beta1.ShootList]()
		r := c.Register("test-data")
		stopFn, err := fixServer(l, c)
		assert.NoError(t, err)
		defer stopFn()

		o := &options{
			socketAddress: socketAddress,
			socketNetwork: socketNetwork,
			writer:        io.Discard,
			Options: &command.Options{
				Logger:  l,
				Context: context.Background(),
			},
		}
		cmd := NewCmd(o)
		o.createdBy = "owner"
		o.outFormat = *output.New(&o.outFormat, output.TableType, "%a", "%e")

		r.Set(&v1beta1.ShootList{
			Items: []v1beta1.Shoot{
				{}, {}, {},
			},
		})

		err = cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("empty output format", func(t *testing.T) {
		c := agent.NewCache[*v1beta1.ShootList]()
		r := c.Register("test-data")
		stopFn, err := fixServer(l, c)
		assert.NoError(t, err)
		defer stopFn()

		o := &options{
			socketAddress: socketAddress,
			socketNetwork: socketNetwork,
			writer:        io.Discard,
			Options: &command.Options{
				Logger:  l,
				Context: context.Background(),
			},
		}
		cmd := NewCmd(o)
		o.createdBy = "owner"
		o.outFormat = *output.New(&o.outFormat, "", "%a", "%e")

		r.Set(&v1beta1.ShootList{
			Items: []v1beta1.Shoot{
				{}, {}, {},
			},
		})

		err = cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("client error", func(t *testing.T) {
		c := agent.NewCache[*v1beta1.ShootList]()
		r := c.Register("test-data")

		o := &options{
			socketAddress: socketAddress,
			socketNetwork: socketNetwork,
			writer:        io.Discard,
			Options: &command.Options{
				Logger:  l,
				Context: context.Background(),
			},
		}
		cmd := NewCmd(o)
		o.createdBy = "owner"
		o.outFormat = *output.New(&o.outFormat, output.TextType, "%a", "%e")

		r.Set(&v1beta1.ShootList{
			Items: []v1beta1.Shoot{
				{}, {}, {},
			},
		})

		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("dial error", func(t *testing.T) {
		c := agent.NewCache[*v1beta1.ShootList]()
		r := c.Register("test-data")

		o := &options{
			socketAddress: "\n",
			writer:        io.Discard,
			Options: &command.Options{
				Logger:  l,
				Context: context.Background(),
			},
		}
		cmd := NewCmd(o)
		o.createdBy = "owner"
		o.outFormat = *output.New(&o.outFormat, output.TextType, "%a", "%e")

		r.Set(&v1beta1.ShootList{
			Items: []v1beta1.Shoot{
				{}, {}, {},
			},
		})

		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("set nil and print error", func(t *testing.T) {
		c := agent.NewCache[*v1beta1.ShootList]()
		r := c.Register("test-data")

		o := &options{
			socketAddress: socketAddress,
			socketNetwork: socketNetwork,
			writer:        io.Discard,
			Options: &command.Options{
				Logger:  l,
				Context: context.Background(),
			},
		}
		cmd := NewCmd(o)
		o.createdBy = "owner"

		r.Set(nil)

		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})
}

func fixServer(l *logrus.Logger, c agent.Cache[*v1beta1.ShootList]) (stop func(), err error) {
	lis, err := agent.NewSocket(socketNetwork, socketAddress)
	if err != nil {
		return nil, err
	}

	grpcServer := googlerpc.NewServer(googlerpc.EmptyServerOption{})
	agentServer := agent.NewServer(&agent.ServerOption{
		GardenerCache: c,
		Logger:        l,
	})
	cloud_agent.RegisterAgentServer(grpcServer, agentServer)

	go grpcServer.Serve(lis)

	return grpcServer.Stop, nil
}