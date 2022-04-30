package testutil

import (
	"context"
	"io"
	"net"

	"github.com/suffiks/suffiks"
	suffiksv1 "github.com/suffiks/suffiks/api/v1"
	"github.com/suffiks/suffiks/base"
	"github.com/suffiks/suffiks/extension"
	"github.com/suffiks/suffiks/extension/protogen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"sigs.k8s.io/yaml"
)

type Suffiks[Ext any] struct {
	extension extension.Extension[Ext]

	ctrl       *base.ExtensionController
	listener   *bufconn.Listener
	grpcServer *grpc.Server
}

func New[Ext any](ext extension.Extension[Ext], spec io.Reader) (*Suffiks[Ext], error) {
	t := &Suffiks[Ext]{
		extension: ext,
	}
	extMgr, err := base.NewExtensionManager(suffiks.CRDFiles, []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithContextDialer(t.dialer),
	})
	if err != nil {
		return nil, err
	}

	var extObj suffiksv1.Extension
	b, _ := io.ReadAll(spec)
	if err := yaml.Unmarshal(b, &extObj); err != nil {
		return nil, err
	}

	extMgr.Add(extObj)
	t.ctrl = base.NewExtensionController(extMgr)
	return t, nil
}

func (t *Suffiks[Ext]) Run(ctx context.Context) error {
	return nil
}

func (t *Suffiks[Ext]) Stop() error {
	if t.grpcServer != nil {
		t.grpcServer.Stop()
	}
	if t.listener != nil {
		return t.listener.Close()
	}
	return nil
}

func (m *Suffiks[Ext]) dialer(context.Context, string) (net.Conn, error) {
	m.init()
	return m.listener.Dial()
}

func (t *Suffiks[Ext]) init() {
	if t.listener != nil {
		return
	}

	t.listener = bufconn.Listen(1024 * 1024)
	t.grpcServer = grpc.NewServer()
	protogen.RegisterExtensionServer(t.grpcServer, extension.NewServer(t.extension, nil))
	go t.grpcServer.Serve(t.listener)
}
