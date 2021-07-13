// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// TODO(bradrydzewski) if the cloud_id is omitted we should
// use the site_id to fetch the cloud_id.
//
//    curl https://droneio.atlassian.net/_edge/tenant_info
//    {"cloudId":"b11a072e-a403-418d-a809-fbf4eb9c434b"}

// Args provides plugin execution arguments.
type Args struct {
	Pipeline

	// Level defines the plugin log level.
	Level string `envconfig:"PLUGIN_LOG_LEVEL"`

	// Atlassian Cloud ID (required)
	CloudID string `envconfig:"PLUGIN_CLOUD_ID"`

	// Atlassian Oauth2 Client ID (required)
	ClientID string `envconfig:"PLUGIN_CLIENT_ID"`

	// Atlassian Oauth2 Client Secret (required)
	ClientSecret string `envconfig:"PLUGIN_CLIENT_SECRET"`

	// Site Name (optional)
	Site string `envconfig:"PLUGIN_INSTANCE"`

	// Project Name (required)
	Project string `envconfig:"PLUGIN_PROJECT"`

	// Pipeline Name (optional)
	Name string `envconfig:"PLUGIN_PIPELINE"`

	// Deployment environment (optional)
	Environment string `envconfig:"PLUGIN_ENVIRONMENT"`

	// Link to deployment (optional)
	Link string `envconfig:"PLUGIN_LINK"`

	// State of the deployment (optional)
	State string `envconfig:"PLUGIN_STATE"`

	// Deployment use only 1 issue instead of multiple (optional)
	SingleIssueDeployment bool `envconfig:"PLUGIN_SINGLE_ISSUE_DEPLOYMENT"`
}

// Exec executes the plugin.
func Exec(ctx context.Context, args Args) error {
	var (
		environ     = toEnvironment(args)
		state       = toState(args)
		version     = toVersion(args)
		deeplink    = toLink(args)
		instance    = args.Site
		issues      = extractIssues(args)
		description = toDescription(args)
	)

	logger := logrus.
		WithField("client_id", args.ClientID).
		WithField("cloud_id", args.CloudID).
		WithField("project_id", args.Project).
		WithField("instance", instance).
		WithField("pipeline", args.Name).
		WithField("environment", environ).
		WithField("state", state).
		WithField("version", version)

	if len(issues) == 0 {
		logger.Debugln("cannot find issue number")
		return errors.New("failed to extract issue number")
	}

	logger = logger.WithField("issue", issues)
	logger.Debugln("successfully extracted issue number")

	payload := Payload{
		Deployments: []*Deployment{
			{
				Deploymentsequencenumber: args.Build.Number,
				Updatesequencenumber:     args.Build.Number,
				Associations: []Association{
					{
						Associationtype: "issueIdOrKeys",
						Values:          issues,
					},
				},
				Displayname: version,
				URL:         deeplink,
				Description: description,
				Lastupdated: time.Now(),
				State:       state,
				Pipeline: JiraPipeline{
					ID:          args.Commit.Author.Email,
					Displayname: args.Commit.Author.Name,
					URL:         deeplink,
				},
				Environment: Environment{
					ID:          environ,
					Displayname: environ,
					Type:        environ,
				},
			},
		},
	}

	if instance != "" {
		logger.Debugln("retrieve cloud id")

		tenant, err := lookupTenant(instance)
		if err != nil {
			logger.WithError(err).
				Errorln("cannot retrieve cloud_id")
			return err
		}
		// HACK: we should avoid mutating args
		args.CloudID = tenant.ID
		logger = logger.WithField("cloud_id", tenant.ID)
		logger.Debugln("successfully retrieved cloud id")
	}

	logger.Debugln("creating token")
	token, err := createToken(args)
	if err != nil {
		logger.Debugln("cannot create token")
		return err
	}

	logger.Infoln("creating deployment")
	response, err := createDeployment(args, payload, token)
	logger.Infoln(response)

	return err
}

// makes an API call to create a token.
func createToken(args Args) (string, error) {
	payload := map[string]string{
		"audience":      "api.atlassian.com",
		"grant_type":    "client_credentials",
		"client_id":     args.ClientID,
		"client_secret": args.ClientSecret,
	}
	endpoint := "https://api.atlassian.com/oauth/token"
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(payload)
	req, err := http.NewRequest("POST", endpoint, buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode > 299 {
		return "", fmt.Errorf("Error code %d", res.StatusCode)
	}
	output := map[string]interface{}{}
	err = json.Unmarshal(out, &output)
	if err != nil {
		return "", err
	}
	return output["access_token"].(string), nil
}

// makes an API call to create a deployment.
func createDeployment(args Args, payload Payload, token string) (string, error) {
	endpoint := fmt.Sprintf("https://api.atlassian.com/jira/deployments/0.1/cloud/%s/bulk", args.CloudID)
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(payload)
	req, err := http.NewRequest("POST", endpoint, buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("From", "noreply@localhost")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return "", fmt.Errorf("Error code %d", res.StatusCode)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(b), err
}

// makes an API call to lookup the cloud ID
func lookupTenant(tenant string) (*Tenant, error) {
	uri := fmt.Sprintf("https://%s.atlassian.net/_edge/tenant_info", tenant)
	res, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return nil, fmt.Errorf("Error code %d", res.StatusCode)
	}
	out := new(Tenant)
	err = json.NewDecoder(res.Body).Decode(out)
	return out, err
}
