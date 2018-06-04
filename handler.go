package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"io"
)

type handler interface {
	Handle(secret secret) error
}

type committer struct {
	ssmiface.SSMAPI
}

func (s *committer) Handle(secret secret) error {
	if _, err := s.PutParameter(makeInput(secret)); err != nil {
		return fmt.Errorf("failed uploading %v: %v", secret.Name, err)
	}

	return nil
}

func makeInput(secret secret) *ssm.PutParameterInput {
	return &ssm.PutParameterInput{
		AllowedPattern: &secret.Pattern,
		Description:    &secret.Description,
		Value:          &secret.Value,
		Overwrite:      aws.Bool(true),
		Type:           aws.String(ssm.ParameterTypeSecureString),
		Name:           &secret.Name,
	}
}

func newCommitter() handler {
	return &committer{ssm.New(session.Must(session.NewSession()))}
}

func newPrinter(writer io.Writer) handler {
	encoder := json.NewEncoder(writer)
	return &printer{
		encoder,
	}
}

type printer struct {
	*json.Encoder
}

func (s *printer) Handle(secret secret) error {
	return s.Encode(secret)
}
