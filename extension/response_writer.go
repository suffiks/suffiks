package extension

import (
	"github.com/suffiks/suffiks/extension/protogen"
)

type grpcWriter interface {
	Send(resp *protogen.Response) error
}

type ResponseWriter struct {
	w grpcWriter
}

func (r *ResponseWriter) AddEnv(name, value string) error {
	return r.w.Send(&protogen.Response{
		OFResponse: &protogen.Response_Env{
			Env: &protogen.KeyValue{
				Name:  name,
				Value: value,
			},
		},
	})
}

func (r *ResponseWriter) AddLabel(name, value string) error {
	return r.w.Send(&protogen.Response{
		OFResponse: &protogen.Response_Label{
			Label: &protogen.KeyValue{
				Name:  name,
				Value: value,
			},
		},
	})
}

func (r *ResponseWriter) AddAnnotation(name, value string) error {
	return r.w.Send(&protogen.Response{
		OFResponse: &protogen.Response_Annotation{
			Annotation: &protogen.KeyValue{
				Name:  name,
				Value: value,
			},
		},
	})
}

// MergePatch is a helper function to add a JSON merge patch to the response.
//
// Using MergePatch heavily requires the extension to know about the underlying Kind it's applied to
// and is only recommended to use if the other helper functions doesn't provide the required result.
func (r *ResponseWriter) MergePatch(b []byte) error {
	return r.w.Send(&protogen.Response{
		OFResponse: &protogen.Response_MergePatch{
			MergePatch: b,
		},
	})
}
