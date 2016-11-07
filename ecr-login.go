package main

import (
	"encoding/base64"

	"os"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"log"
	"C"
)

type Auth struct {
	Token         string
	User          string
	Pass          string
	ProxyEndpoint string
	ExpiresAt     time.Time
}

// error handler
func check(e error) {
	if e != nil {
		panic(e.Error())
	}
}

// default template prints docker login command
const DEFAULT_TEMPLATE = `{{range .}}docker login -u {{.User}} -p {{.Pass}} -e none {{.ProxyEndpoint}}
{{end}}`

// load template from file or use default
func getTemplate() *template.Template {
	var tmpl *template.Template
	var err error

	file, exists := os.LookupEnv("TEMPLATE")

	if exists {
		tmpl, err = template.ParseFiles(file)
	} else {
		tmpl, err = template.New("default").Parse(DEFAULT_TEMPLATE)
	}

	check(err)
	return tmpl
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("ecr-login error: %v\n",r)
		}
	}()
	var registryIds []*string

	registries, exists := os.LookupEnv("REGISTRIES")

	if exists {
		for _, registry := range strings.Split(registries, ",") {
			registryIds = append(registryIds, aws.String(registry))
		}
	}

	svc := ecr.New(session.New())

	// get tokens for multiple registries
	params := &ecr.GetAuthorizationTokenInput{
		RegistryIds: registryIds,
	}

	resp, err := svc.GetAuthorizationToken(params)
	check(err)

	// fields to send to template
	fields := make([]Auth, len(resp.AuthorizationData))
	for i, auth := range resp.AuthorizationData {

		// extract base64 token
		data, err := base64.StdEncoding.DecodeString(*auth.AuthorizationToken)
		check(err)

		// extract username and password
		token := strings.SplitN(string(data), ":", 2)

		// object to pass to template
		fields[i] = Auth{
			Token:         *auth.AuthorizationToken,
			User:          token[0],
			Pass:          token[1],
			ProxyEndpoint: *(auth.ProxyEndpoint),
			ExpiresAt:     *(auth.ExpiresAt),
		}
	}

	// run the template
	err = getTemplate().Execute(os.Stdout, fields)
	check(err)
}
