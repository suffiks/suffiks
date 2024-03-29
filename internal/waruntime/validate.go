package waruntime

import (
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type ValidationError struct {
	Functions []funcDecl
	Has       []funcDecl
}

func (e *ValidationError) Error() string {
	sb := strings.Builder{}
	sb.WriteString("missing or invalid functions:")
	for _, f := range e.Functions {
		sb.WriteString("\n\t" + f.String())
	}

	if len(e.Has) > 0 {
		sb.WriteString("\nhas:")
		for _, f := range e.Has {
			sb.WriteString("\n\t" + f.String())
		}
	}

	return sb.String()
}

type funcDecl struct {
	name    string
	args    []api.ValueType
	returns []api.ValueType
}

func (f funcDecl) String() string {
	sb := strings.Builder{}
	sb.WriteString(f.name + "(")
	for i, arg := range f.args {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(api.ValueTypeName(arg))
	}
	sb.WriteString(") ")
	for i, ret := range f.returns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(api.ValueTypeName(ret))
	}

	return sb.String()
}

var requiredFunctions = []funcDecl{
	{
		name: "Sync",
	},
	{
		name:    "Delete",
		returns: []api.ValueType{api.ValueTypeI64},
	},
	{
		name:    "Defaulting",
		returns: []api.ValueType{api.ValueTypeI64},
	},
	{
		name: "Validate",
		args: []api.ValueType{api.ValueTypeI32},
	},
	{
		name:    "malloc",
		args:    []api.ValueType{api.ValueTypeI32},
		returns: []api.ValueType{api.ValueTypeI32},
	},
	{
		name: "free",
		args: []api.ValueType{api.ValueTypeI32},
	},
}

func validate(module wazero.CompiledModule) error {
	funcs := module.ExportedFunctions()

	// for name, f := range funcs {
	// 	fmt.Print(name + "(")
	// 	for i, arg := range f.ParamTypes() {
	// 		if i > 0 {
	// 			fmt.Print(", ")
	// 		}
	// 		fmt.Print(api.ValueTypeName(arg))
	// 	}
	// 	fmt.Print(") ")
	// 	for i, ret := range f.ResultTypes() {
	// 		if i > 0 {
	// 			fmt.Print(", ")
	// 		}
	// 		fmt.Print(api.ValueTypeName(ret))
	// 	}
	// 	fmt.Println()
	// }

	err := &ValidationError{}

	for _, sig := range requiredFunctions {
		f, ok := funcs[sig.name]
		if !ok {
			err.Functions = append(err.Functions, sig)
			continue
		}

		if !equalTypes(f.ResultTypes(), sig.returns) {
			err.Functions = append(err.Functions, sig)
			continue
		}
		if !equalTypes(f.ParamTypes(), sig.args) {
			err.Functions = append(err.Functions, sig)
			continue
		}

	}

	if len(err.Functions) > 0 {
		for name, f := range funcs {
			err.Has = append(err.Has, funcDecl{
				name:    name,
				args:    f.ParamTypes(),
				returns: f.ResultTypes(),
			})
		}

		return err
	}

	return nil
}

func equalTypes(a, b []api.ValueType) bool {
	if len(a) != len(b) {
		return false
	}

	for i, t := range a {
		if t != b[i] {
			return false
		}
	}

	return true
}
