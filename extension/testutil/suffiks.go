package testutil

import (
	"context"
	"io"
	"net"

	"github.com/suffiks/suffiks"
	suffiksv1 "github.com/suffiks/suffiks/apis/suffiks/v1"
	"github.com/suffiks/suffiks/extension"
	"github.com/suffiks/suffiks/extension/protogen"
	controller "github.com/suffiks/suffiks/internal/controllers"
	intexternal "github.com/suffiks/suffiks/internal/extension"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"sigs.k8s.io/yaml"
)

type Suffiks[Ext any] struct {
	extension extension.Extension[Ext]

	ctrl       *controller.ExtensionController
	listener   *bufconn.Listener
	grpcServer *grpc.Server
}

func New[Ext any](ext extension.Extension[Ext], spec io.Reader) (*Suffiks[Ext], error) {
	t := &Suffiks[Ext]{
		extension: ext,
	}
	extMgr, err := intexternal.NewExtensionManager(suffiks.CRDFiles, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
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
	t.ctrl = controller.NewExtensionController(extMgr)
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
