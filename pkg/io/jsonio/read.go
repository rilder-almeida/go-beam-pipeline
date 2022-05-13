package jsonio

import (
	"reflect"

	"github.com/apache/beam/sdks/v2/go/pkg/beam"
	"github.com/apache/beam/sdks/v2/go/pkg/beam/io/textio"

	"github.com/johannaojeling/go-beam-pipeline/pkg/io/stringio"
)

func Read(scope beam.Scope, inputPath string, elemType reflect.Type) beam.PCollection {
	scope = scope.Scope("Read from json")
	col := textio.ReadSdf(scope, inputPath)
	encoded := beam.ParDo(scope, &stringio.EncodeFn{}, col)
	return beam.ParDo(
		scope,
		&UnMarshalJsonFn{Type: beam.EncodedType{T: elemType}},
		encoded,
		beam.TypeDefinition{Var: beam.XType, T: elemType},
	)
}
