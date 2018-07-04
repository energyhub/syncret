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

// return a new syncer which commits values to the SSM api
func newCommitter() syncer {
	return &committer{ssm.New(session.Must(session.NewSession()))}
}

// return a new syncer which writes secret metadata (not the value itself) to the provided writer
func newPrinter(writer io.Writer) syncer {
	encoder := json.NewEncoder(writer)
	return &printer{
		encoder,
	}
}

// the "real" syncer -- commits the value to the SSM api
type committer struct {
	ssmiface.SSMAPI
}

func (s *committer) Sync(secret secret) error {
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
		Overwrite:      aws.Bool(true),                            // always overwrite
		Type:           aws.String(ssm.ParameterTypeSecureString), // always secure
		Name:           &secret.Name,
	}
}

// a "syncer" which outputs secrets (excluding the actual secret value) as JSON
// note that redaction behavior relies on `secret.Value` having a json name tag of '-' to
// redact the value
type printer struct {
	*json.Encoder
}

func (s *printer) Sync(secret secret) error {
	return s.Encode(secret)
}
