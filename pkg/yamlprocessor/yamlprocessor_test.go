// Copyright 2020 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package yamlprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetValue(t *testing.T) {
	testcases := []struct {
		name    string
		yml     string
		path    string
		want    interface{}
		wantErr bool
	}{
		{
			name:    "empty yaml given",
			yml:     "",
			path:    "$.foo",
			wantErr: true,
		},
		{
			name:    "empty path given",
			yml:     "foo: bar",
			path:    "",
			wantErr: true,
		},
		{
			name:    "wrong path given",
			yml:     "foo: bar",
			path:    "wrong",
			wantErr: true,
		},
		{
			name:    "wrong yaml given",
			yml:     "::",
			path:    "$.foo",
			wantErr: true,
		},
		{
			name:    "lack of root element",
			yml:     "foo: bar",
			path:    "foo",
			wantErr: true,
		},
		{
			name:    "given a string path",
			yml:     "foo: bar",
			path:    "$.foo",
			want:    "bar",
			wantErr: false,
		},
		{
			name:    "given a bool path",
			yml:     "foo: true",
			path:    "$.foo",
			want:    true,
			wantErr: false,
		},
		{
			name:    "given a uint64 path",
			yml:     "foo: 1",
			path:    "$.foo",
			want:    uint64(1),
			wantErr: false,
		},
		{
			name:    "given a int64 path",
			yml:     "foo: -1",
			path:    "$.foo",
			want:    int64(-1),
			wantErr: false,
		},
		{
			name:    "given a float64 path",
			yml:     "foo: 1.5",
			path:    "$.foo",
			want:    1.5,
			wantErr: false,
		},
		{
			name: "given an array path",
			yml: `
foo:
- bar: 1`,
			path:    "$.foo[0].bar",
			want:    uint64(1),
			wantErr: false,
		},
		{
			name: "given an entire array path",
			yml: `
foo:
- bar: 1
- baz: 2`,
			path:    "$.foo",
			want:    []interface{}{map[string]interface{}{"bar": uint64(1)}, map[string]interface{}{"baz": uint64(2)}},
			wantErr: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetValue([]byte(tc.yml), tc.path)
			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestReplaceValue(t *testing.T) {
	testcases := []struct {
		name    string
		yml     string
		path    string
		value   string
		want    []byte
		wantErr bool
	}{
		{
			name:    "empty yaml given",
			yml:     "",
			path:    "$.foo",
			value:   "new-text",
			wantErr: true,
		},
		{
			name:    "empty path given",
			yml:     "foo: bar",
			path:    "",
			value:   "new-text",
			wantErr: true,
		},
		{
			name:    "wrong path given",
			yml:     "foo: bar",
			path:    "wrong",
			value:   "new-text",
			wantErr: true,
		},
		{
			name:    "wrong yaml given",
			yml:     "::",
			path:    "$.foo",
			value:   "new-text",
			wantErr: true,
		},
		{
			name:    "empty value given",
			yml:     "foo: bar",
			path:    "$.foo",
			value:   "",
			want:    []byte("foo: "),
			wantErr: false,
		},
		{
			name:    "valid value given",
			yml:     "foo: bar",
			path:    "$.foo",
			value:   "new-text",
			want:    []byte("foo: new-text"),
			wantErr: false,
		},
		{
			name: "valid value with comment given",
			yml: `# comments
foo: bar`,
			path:  "$.foo",
			value: "new-text",
			want: []byte(`# comments
foo: new-text`),
			wantErr: false,
		},
		{
			name: "array in block style",
			yml: `foo:
  - bar
  - baz`,
			path:  "$.foo[0]",
			value: "new-text",
			want: []byte(`foo:
  - new-text
  - baz`),
			wantErr: false,
		},
		{
			name:    "array in flow style",
			yml:     `foo: [bar, baz]`,
			path:    "$.foo[0]",
			value:   "new-text",
			want:    []byte(`foo: [new-text, baz]`),
			wantErr: false,
		},
		{
			name: "there is an useless blank line",
			yml: `
foo:
  - bar
  - baz`,
			path:  "$.foo[0]",
			value: "new-text",
			want: []byte(`foo:
  - new-text
  - baz`),
			wantErr: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReplaceValue([]byte(tc.yml), tc.path, tc.value)
			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.want, got)
		})
	}
}
