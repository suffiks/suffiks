package waruntime

import (
	"context"
	"encoding/json"

	"github.com/davecgh/go-spew/spew"
	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/tetratelabs/wazero/api"
	"google.golang.org/protobuf/proto"
)

func addEnv(_ context.Context, m api.Module, ptr, size uint32) {
	b, ok := m.Memory().Read(ptr, size)
	if !ok {
		panic("failed to read memory")
	}

	kv := &protogen.KeyValue{}
	if err := proto.Unmarshal(b, kv); err != nil {
		panic("addEnvasdfsadfasfs: " + err.Error())
	}

	spew.Dump(kv)
}

func getOwner(ctx context.Context, m api.Module) uint32 {
	owner := protogen.Owner{
		Name: "suffiksownername",
	}

	b, err := proto.Marshal(&owner)
	if err != nil {
		panic("getOwner: " + err.Error())
	}

	res, err := m.ExportedFunction("Malloc").Call(ctx, uint64(len(b)))
	if err != nil {
		panic("getOwner: " + err.Error())
	}

	ptr := res[0]
	if ok := m.Memory().Write(uint32(ptr), b); !ok {
		panic("getOwner: unable to write to memory")
	}

	ptrAndSize := uint32(ptr)<<16 | uint32(len(b))

	return ptrAndSize
}

func getSpec(ctx context.Context, m api.Module) uint32 {
	b, _ := json.Marshal(map[string]any{
		"ingresses": []map[string]any{
			{
				"host": "suffiks.com",
			},
			{
				"host":  "suffiks.com",
				"paths": []string{"/test"},
			},
		},
	})

	res, err := m.ExportedFunction("Malloc").Call(ctx, uint64(len(b)))
	if err != nil {
		panic("getOwner: " + err.Error())
	}

	ptr := res[0]
	if ok := m.Memory().Write(uint32(ptr), b); !ok {
		panic("getOwner: unable to write to memory")
	}

	ptrAndSize := uint32(ptr)<<16 | uint32(len(b))

	return ptrAndSize
}
