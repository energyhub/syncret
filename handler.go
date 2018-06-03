package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
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
	input := &ssm.PutParameterInput{
		AllowedPattern: &secret.Pattern,
		Description:    &secret.Description,
		Value:          &secret.Value,
		Type:           aws.String(ssm.ParameterTypeSecureString),
		Name:           &secret.Name,
	}

	if _, err := s.PutParameter(input); err != nil {
		return fmt.Errorf("failed uploading %v: %v", secret.Name, err)
	}

	return nil
}

func newPrinter(writer io.Writer) *printer {
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
