package main

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func Test_envMap(t *testing.T) {
	type args struct {
		environ []string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			"manages multiple equals",
			args{[]string{
				"value=blah=blah",
			}},
			map[string]string{
				"value": "blah=blah",
			},
		},
		{
			"overwrites",
			args{[]string{
				"value=blah",
				"value=blah=blah",
			}},
			map[string]string{
				"value": "blah=blah",
			},
		},
		{
			"multi value",
			args{[]string{
				"value=blah",
				"value1=blah=blah",
			}},
			map[string]string{
				"value":  "blah",
				"value1": "blah=blah",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := envMap(tt.args.environ); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("envMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockLoader struct {
	secrets []secret
	e       error
}

func (m *mockLoader) LoadAll(paths []string) ([]secret, error) {
	return m.secrets, m.e
}

type mockHandler struct {
	handler
}

func Test_run(t *testing.T) {
	type args struct {
		loader  loader
		in      io.Reader
		handler handler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"no paths no errors",
			args{
				loader:  &mockLoader{},
				handler: mockHandler{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := run(tt.args.loader, tt.args.handler, []string{}); (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getPaths(t *testing.T) {
	type args struct {
		in   io.Reader
		args []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"prefers args",
			args{
				nil,
				[]string{"from args"},
			},
			[]string{"from args"},
		},
		{
			"uses buf when args empty",
			args{
				bytes.NewBufferString("from buffer"),
				nil,
			},
			[]string{"from buffer"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPaths(tt.args.in, tt.args.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPaths() = %v, want %v", got, tt.want)
			}
		})
	}
}
