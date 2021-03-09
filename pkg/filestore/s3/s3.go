// Copyright 2021 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"

	"github.com/pipe-cd/pipe/pkg/filestore"
)

type Store struct {
	client          *s3.Client
	bucket          string
	profile         string
	credentialsFile string
	roleARN         string
	tokenFile       string

	logger *zap.Logger
}

type Option func(*Store)

func WithLogger(logger *zap.Logger) Option {
	return func(s *Store) {
		s.logger = logger.Named("s3")
	}
}

func WithCredentialsFile(path, profile string) Option {
	return func(s *Store) {
		s.profile = profile
		s.credentialsFile = path
	}
}

func WithTokenFile(roleARN, path string) Option {
	return func(s *Store) {
		s.roleARN = roleARN
		s.tokenFile = path
	}
}

func NewStore(ctx context.Context, region, bucket string, opts ...Option) (*Store, error) {
	if region == "" {
		return nil, fmt.Errorf("region is required field")
	}
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required field")
	}

	s := &Store{
		bucket: bucket,
		logger: zap.NewNop(),
	}
	for _, opt := range opts {
		opt(s)
	}

	var optFns []func(*config.LoadOptions) error
	optFns = append(optFns, config.WithRegion(region))
	if s.credentialsFile != "" {
		optFns = append(optFns, config.WithSharedCredentialsFiles([]string{s.credentialsFile}))
	}
	if s.tokenFile != "" && s.roleARN != "" {
		optFns = append(optFns, config.WithWebIdentityRoleCredentialOptions(func(v *stscreds.WebIdentityRoleOptions) {
			v.RoleARN = s.roleARN
			v.TokenRetriever = stscreds.IdentityTokenFile(s.tokenFile)
		}))
	}

	// When you initialize an aws.Config instance using config.LoadDefaultConfig, the SDK uses its default credential chain to find AWS credentials.
	// This default credential chain looks for credentials in the following order:
	//
	// 1. Environment variables.
	//   1. Static Credentials (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN)
	//   2. Web Identity Token (AWS_WEB_IDENTITY_TOKEN_FILE)
	// 2. Shared configuration files.
	//   1. SDK defaults to credentials file under .aws folder that is placed in the home folder on your computer.
	//   2. SDK defaults to config file under .aws folder that is placed in the home folder on your computer.
	// 3. If your application uses an ECS task definition or RunTask API operation, IAM role for tasks.
	// 4. If your application is running on an Amazon EC2 instance, IAM role for Amazon EC2.
	// ref: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load config to create s3 client: %w", err)
	}
	s.client = s3.NewFromConfig(cfg)

	return s, nil
}

func (s *Store) NewReader(ctx context.Context, path string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}
	out, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *Store) GetObject(ctx context.Context, path string) (object filestore.Object, err error) {
	rc, err := s.NewReader(ctx, path)
	if err != nil {
		return
	}
	defer func() {
		if err := rc.Close(); err != nil {
			s.logger.Error("failed to close object reader")
		}
	}()

	content, err := ioutil.ReadAll(rc)
	if err != nil {
		return
	}

	object.Path = path
	object.Content = content
	object.Size = int64(len(content))
	return
}

func (s *Store) PutObject(ctx context.Context, path string, content []byte) error {
	input := &s3.PutObjectInput{
		Body:   bytes.NewReader(content),
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}
	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) ListObjects(ctx context.Context, prefix string) ([]filestore.Object, error) {
	var objects []filestore.Object
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get list objects: %w", err)
		}
		for _, obj := range page.Contents {
			objects = append(objects, filestore.Object{
				Path:    aws.ToString(obj.Key),
				Size:    obj.Size,
				Content: []byte{},
			})
		}
	}
	return objects, nil
}

func (s *Store) Close() error {
	// aws client does not provide the way to close a connection via sdk
	return nil
}
