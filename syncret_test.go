package main

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"fmt"
)

type mockLoader struct {
	secrets []secret
	e       error
}

func (m *mockLoader) LoadAll(paths []string) ([]secret, error) {
	return m.secrets, m.e
}

type mockSyncer struct {
	errors map[string]error
	synced []string
}

func (m *mockSyncer) Sync(s secret) error {
	if e := m.errors[s.Name]; e != nil {
		return e
	}
	m.synced = append(m.synced, s.Name)
	return nil
}

func Test_run(t *testing.T) {
	type args struct {
		loader loader
		in     io.Reader
		syncer *mockSyncer
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"no paths no errors",
			args{
				loader: &mockLoader{},
				syncer: &mockSyncer{
					errors: make(map[string]error),
					synced: make([]string, 0),
				},
			},
			make([]string, 0),
			false,
		},
		{
			"errs on loader error",
			args{
				loader: &mockLoader{
					e: fmt.Errorf("blah blah"),
				},
				syncer: &mockSyncer{
					errors: make(map[string]error),
					synced: make([]string, 0),
				},
			},
			make([]string, 0),
			true,
		},
		{
			"propagates syncer error",
			args{
				loader: &mockLoader{
					secrets: []secret{
						{
							Name: "hihihihi",
						},
					},
				},
				syncer: &mockSyncer{
					errors: map[string]error{
						"hihihihi": fmt.Errorf("found an error"),
					},
					synced: make([]string, 0),
				},
			},
			make([]string, 0),
			true,
		},
		{
			"partial sync",
			args{
				loader: &mockLoader{
					secrets: []secret{
						{
							Name: "synced",
						},
						{
							Name: "fails",
						},
						{
							Name: "not hit",
						},
					},
				},
				syncer: &mockSyncer{
					errors: map[string]error{
						"fails": fmt.Errorf("found an error"),
					},
					synced: make([]string, 0),
				},
			},
			[]string{"synced"},
			true,
		},
		{
			"all sync",
			args{
				loader: &mockLoader{
					secrets: []secret{
						{
							Name: "synced",
						},
					},
				},
				syncer: &mockSyncer{
					errors: make(map[string]error),
					synced: make([]string, 0),
				},
			},
			[]string{"synced"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := run(tt.args.loader, tt.args.syncer, []string{}); (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.want, tt.args.syncer.synced) {
				t.Errorf("synced = %v, want %v", tt.args.syncer.synced, tt.want)
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
