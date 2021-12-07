package main

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type SecretAccessor struct {
	projectName string
	client      *secretmanager.Client
}

func NewSecretAccessor(projectName string) (SecretAccessor, error) {
	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return SecretAccessor{}, fmt.Errorf("failed to create secretmanager client: %v", err)
	}

	return SecretAccessor{
		projectName: projectName,
		client:      client,
	}, nil
}

type secretAccess func(name string) (string, error)

func (sa SecretAccessor) GetAllVariables(flags map[string]*string) error {

	if err := sa.PopulateFlags(flags, sa.accessSecretVersion); err != nil {
		return err
	}

	return nil
}

func (sa SecretAccessor) PopulateFlags(flags map[string]*string, accessSecrets secretAccess) error {
	var errs []error
	for k, v := range flags {
		secretName := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", sa.projectName, k)
		result, err := accessSecrets(secretName)
		if err != nil {
			errs = append(errs, err)
		}
		*v = result
	}
	if len(errs) > 0 {
		return fmt.Errorf("get secrets failed: %v", errs)
	}

	return nil
}

// accessSecretVersion accesses the payload for the given secret version if one
// exists. The version can be a version number as a string (e.g. "5") or an
// alias (e.g. "latest").
func (sa SecretAccessor) accessSecretVersion(name string) (string, error) {
	// name := "projects/my-project/secrets/my-secret/versions/5"
	// name := "projects/my-project/secrets/my-secret/versions/latest"

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	// Call the API.
	ctx := context.Background()
	result, err := sa.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %v", err)
	}

	return string(result.Payload.Data), nil
}
