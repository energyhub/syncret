package main

import (
	"fmt"
	"reflect"
	"testing"

	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

type MockClient struct {
	ssmiface.SSMAPI
	error error
}

func (c *MockClient) PutParameter(input *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
	return nil, c.error
}

func Test_committer_Handle(t *testing.T) {
	type args struct {
		secret secret
	}

	tests := []struct {
		name    string
		s       *committer
		args    args
		wantErr bool
	}{
		{
			name:    "propagates error",
			s:       &committer{&MockClient{error: fmt.Errorf("my error")}},
			wantErr: true,
		},
		{
			name: "no error is successful",
			s:    &committer{&MockClient{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.Handle(tt.args.secret); (err != nil) != tt.wantErr {
				t.Errorf("committer.Handle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_makeInput(t *testing.T) {
	tests := []struct {
		name   string
		secret secret
		want   *ssm.PutParameterInput
	}{
		{
			"all fields",
			secret{
				"/blah/blah/hi",
				"secret value",
				"I am a description",
				"^.*$",
			},
			&ssm.PutParameterInput{
				AllowedPattern: aws.String("^.*$"),
				Description:    aws.String("I am a description"),
				Name:           aws.String("/blah/blah/hi"),
				Overwrite:      aws.Bool(true),
				Type:           aws.String(ssm.ParameterTypeSecureString),
				Value:          aws.String("secret value"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeInput(tt.secret); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_printer_Handle(t *testing.T) {
	expected := "{\"name\":\"hi\"}\n"
	buf := new(bytes.Buffer)
	newPrinter(buf).Handle(secret{
		Name:  "hi",
		Value: "should be suppressed",
	})

	if expected != buf.String() {
		t.Errorf("Expected %v; got %v", expected, buf.String())
	}
}
