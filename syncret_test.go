package main

import (
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
			},},
			map[string]string{
				"value": "blah=blah",
			},
		},
		{
			"overwrites",
			args{[]string{
				"value=blah",
				"value=blah=blah",
			},},
			map[string]string{
				"value": "blah=blah",
			},
		},
		{
			"multi value",
			args{[]string{
				"value=blah",
				"value1=blah=blah",
			},},
			map[string]string{
				"value": "blah",
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
