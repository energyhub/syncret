package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"io"
	"text/template"
)

var secretTmpl = template.Must(template.New("secret").Parse("{{.Name}} | " +
	"{{if .Description}}{{.Description}}{{else}}No description{{end}} | " +
	"{{if .Pattern}}{{.Pattern}}{{else}}No pattern{{end}}\n"))

type Handler interface {
	Handle(secret secret) error
}

type Committer struct {
	ssmiface.SSMAPI
}

func (s *Committer) Handle(secret secret) error {
	input := &ssm.PutParameterInput{
		AllowedPattern: &secret.Pattern,
		Description:    &secret.Description,
		Value:          &secret.Value,
		Name:           &secret.Name,
	}

	if _, err := s.PutParameter(input); err != nil {
		return fmt.Errorf("failed uploading %v: %v", secret.Name, err)
	}

	return nil
}

func NewPrinter(writer io.Writer) *Printer {
	encoder := json.NewEncoder(writer)
	return &Printer{
		encoder,
	}
}

type Printer struct {
	*json.Encoder
}

func (s *Printer) Handle(secret secret) error {
	return s.Encode(secret)
}
