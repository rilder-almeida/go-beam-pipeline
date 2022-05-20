package redisio

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/apache/beam/sdks/v2/go/pkg/beam"
	"github.com/apache/beam/sdks/v2/go/pkg/beam/testing/ptest"
	"github.com/stretchr/testify/assert"

	"github.com/johannaojeling/go-beam-pipeline/pkg/internal/testutils/redis"
)

func TestWrite(t *testing.T) {
	testCases := []struct {
		reason     string
		expiration time.Duration
		batchSize  int
		fieldKey   string
		elemType   reflect.Type
		input      []interface{}
		keyPrefix  string
		expected   map[string]string
	}{
		{
			reason:   "Should extract key and write to Redis from PCollection of type entry",
			fieldKey: "Field1",
			elemType: reflect.TypeOf(entry{}),
			input: []interface{}{
				entry{
					Field1: "val1",
					Field2: 1,
				},
				entry{
					Field1: "val2",
					Field2: 2,
				},
			},
			keyPrefix: "val*",
			expected: map[string]string{
				"val1": `{"field1":"val1","field2":1}`,
				"val2": `{"field1":"val2","field2":2}`,
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Test %d: %s", i, tc.reason), func(t *testing.T) {
			miniRedis, err := miniredis.Run()
			if err != nil {
				t.Fatalf("error initializing Miniredis: %v", err)
			}
			defer miniRedis.Close()

			address := miniRedis.Addr()
			url := fmt.Sprintf("redis://%s/0", address)

			beam.Init()
			pipeline, scope := beam.NewPipelineWithRoot()

			col := beam.Create(scope, tc.input...)
			Write(scope, url, tc.expiration, tc.batchSize, tc.fieldKey, col)

			ptest.RunAndValidate(t, pipeline)

			actual, err := redis.GetEntries(url, tc.keyPrefix)
			if err != nil {
				t.Fatalf("error getting values: %v", err)
			}

			assert.Equal(t, tc.expected, actual, "Entries should match")
		})
	}
}

func TestWriteKV(t *testing.T) {
	testCases := []struct {
		reason     string
		expiration time.Duration
		batchSize  int
		elemType   reflect.Type
		unpackDoFn interface{}
		input      []interface{}
		keyPrefix  string
		expected   map[string]string
	}{
		{
			reason:     "Should write to Redis from PCollection of type KV<string, string>",
			elemType:   reflect.TypeOf(stringKV{}),
			unpackDoFn: unpackStringKV,
			input: []interface{}{
				stringKV{
					Key:   "skey1",
					Value: "val1",
				},
				stringKV{
					Key:   "skey2",
					Value: "val2",
				},
			},
			keyPrefix: "skey*",
			expected: map[string]string{
				"skey1": "val1",
				"skey2": "val2",
			},
		},
		{
			reason:     "Should write to Redis from PCollection of type KV<string, entry>",
			elemType:   reflect.TypeOf(entryKV{}),
			unpackDoFn: unpackEntryKV,
			input: []interface{}{
				entryKV{
					Key: "ekey1",
					Value: entry{
						Field1: "val1",
						Field2: 1,
					},
				},
				entryKV{
					Key: "ekey2",
					Value: entry{
						Field1: "val2",
						Field2: 2,
					},
				},
			},
			keyPrefix: "ekey*",
			expected: map[string]string{
				"ekey1": `{"field1":"val1","field2":1}`,
				"ekey2": `{"field1":"val2","field2":2}`,
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Test %d: %s", i, tc.reason), func(t *testing.T) {
			miniRedis, err := miniredis.Run()
			if err != nil {
				t.Fatalf("error initializing Miniredis: %v", err)
			}
			defer miniRedis.Close()

			address := miniRedis.Addr()
			url := fmt.Sprintf("redis://%s/0", address)

			beam.Init()
			pipeline, scope := beam.NewPipelineWithRoot()

			col := beam.Create(scope, tc.input...)
			unpacked := beam.ParDo(
				scope,
				tc.unpackDoFn,
				col,
				beam.TypeDefinition{Var: beam.XType, T: tc.elemType},
			)
			WriteKV(scope, url, tc.expiration, tc.batchSize, unpacked)

			ptest.RunAndValidate(t, pipeline)

			actual, err := redis.GetEntries(url, tc.keyPrefix)
			if err != nil {
				t.Fatalf("error getting values: %v", err)
			}

			assert.Equal(t, tc.expected, actual, "Entries should match")
		})
	}
}

type stringKV struct {
	Key   string
	Value string
}

func unpackStringKV(kv stringKV) (string, string) {
	return kv.Key, kv.Value
}

type entry struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

func (entry entry) MarshalBinary() ([]byte, error) {
	return json.Marshal(entry)
}

type entryKV struct {
	Key   string
	Value entry
}

func unpackEntryKV(kv entryKV) (string, entry) {
	return kv.Key, kv.Value
}
