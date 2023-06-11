package extension

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	suffiksv1 "github.com/suffiks/suffiks/api/suffiks/v1"
	"github.com/suffiks/suffiks/extension/protogen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestNewExtensionManager(t *testing.T) {
	expectedTargets := []suffiksv1.Target{
		"Application",
		"Work",
	}

	files := os.DirFS("./testdata")

	listener := &mockGRPCListener{}
	mgr, err := NewExtensionManager(context.Background(), files, []grpc.DialOption{
		grpc.WithContextDialer(listener.Dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	if err != nil {
		t.Fatal(err)
	}

	targets := []suffiksv1.Target{}
	for t := range mgr.spec {
		targets = append(targets, t)
	}

	cmpOpts := []cmp.Option{
		cmpopts.SortSlices(func(a, b suffiksv1.Target) bool {
			return strings.Compare(string(a), string(b)) < 0
		}),
	}
	if !cmp.Equal(expectedTargets, targets, cmpOpts...) {
		t.Error(cmp.Diff(expectedTargets, targets, cmpOpts...))
	}

	ext := suffiksv1.Extension{
		Spec: suffiksv1.ExtensionSpec{
			Targets: []suffiksv1.Target{"Application"},
			Controller: suffiksv1.ControllerSpec{
				GRPC: &suffiksv1.ExtensionGRPCController{},
			},
			OpenAPIV3Schema: runtime.RawExtension{
				Raw: []byte(`{"type":"object","properties":{"foo":{"type":"string"}}}`),
			},
		},
	}
	if err := mgr.Add(ext); err != nil {
		t.Fatal(err)
	}

	if len(mgr.ExtensionsFor("Application")) != 1 {
		t.Error("Expected 1 extension for Application")
	}

	if len(mgr.ExtensionsFor("Work")) != 0 {
		t.Error("Expected 0 extension for Work")
	}

	if err := mgr.Remove(&ext); err != nil {
		t.Fatal(err)
	}

	if len(mgr.ExtensionsFor("Application")) != 0 {
		t.Error("Expected 0 extension for Application")
	}

	if len(mgr.ExtensionsFor("Work")) != 0 {
		t.Error("Expected 0 extension for Work")
	}
}

type mockGRPCListener struct {
	Server protogen.ExtensionServer

	Dials      int
	listener   *bufconn.Listener
	grpcServer *grpc.Server
}

func (m *mockGRPCListener) Stop() {
	m.grpcServer.Stop()
}

func (m *mockGRPCListener) Dialer(context.Context, string) (net.Conn, error) {
	m.init()

	m.Dials++
	return m.listener.Dial()
}

func (m *mockGRPCListener) init() {
	if m.listener != nil {
		return
	}

	m.listener = bufconn.Listen(1024 * 1024)
	m.grpcServer = grpc.NewServer()
	protogen.RegisterExtensionServer(m.grpcServer, m.Server)
	if err := m.grpcServer.Serve(m.listener); err != nil {
		panic(err)
	}
}
