// Copyright (c) 2018 Ashley Jeffs
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package writer

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Jeffail/benthos/lib/log"
	"github.com/Jeffail/benthos/lib/message"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/types"
	sess "github.com/Jeffail/benthos/lib/util/aws/session"
	"github.com/Jeffail/benthos/lib/util/text"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

//------------------------------------------------------------------------------

// AmazonS3Config contains configuration fields for the AmazonS3 output type.
type AmazonS3Config struct {
	sess.Config `json:",inline" yaml:",inline"`
	Bucket      string `json:"bucket" yaml:"bucket"`
	Path        string `json:"path" yaml:"path"`
	ContentType string `json:"content_type" yaml:"content_type"`
	Timeout     string `json:"timeout" yaml:"timeout"`
}

// NewAmazonS3Config creates a new Config with default values.
func NewAmazonS3Config() AmazonS3Config {
	return AmazonS3Config{
		Config:      sess.NewConfig(),
		Bucket:      "",
		Path:        "${!count:files}-${!timestamp_unix_nano}.txt",
		ContentType: "application/octet-stream",
		Timeout:     "5s",
	}
}

//------------------------------------------------------------------------------

// AmazonS3 is a benthos writer.Type implementation that writes messages to an
// Amazon S3 bucket.
type AmazonS3 struct {
	conf AmazonS3Config

	path *text.InterpolatedString

	session  *session.Session
	uploader *s3manager.Uploader
	timeout  time.Duration

	log   log.Modular
	stats metrics.Type
}

// NewAmazonS3 creates a new Amazon S3 bucket writer.Type.
func NewAmazonS3(
	conf AmazonS3Config,
	log log.Modular,
	stats metrics.Type,
) (*AmazonS3, error) {
	var timeout time.Duration
	if tout := conf.Timeout; len(tout) > 0 {
		var err error
		if timeout, err = time.ParseDuration(tout); err != nil {
			return nil, fmt.Errorf("failed to parse timeout period string: %v", err)
		}
	}
	return &AmazonS3{
		conf:    conf,
		log:     log,
		stats:   stats,
		path:    text.NewInterpolatedString(conf.Path),
		timeout: timeout,
	}, nil
}

// Connect attempts to establish a connection to the target S3 bucket.
func (a *AmazonS3) Connect() error {
	if a.session != nil {
		return nil
	}

	sess, err := a.conf.GetSession()
	if err != nil {
		return err
	}

	a.session = sess
	a.uploader = s3manager.NewUploader(sess)

	a.log.Infof("Uploading message parts as objects to Amazon S3 bucket: %v\n", a.conf.Bucket)
	return nil
}

// Write attempts to write message contents to a target S3 bucket as files.
func (a *AmazonS3) Write(msg types.Message) error {
	if a.session == nil {
		return types.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(
		aws.BackgroundContext(), a.timeout,
	)
	defer cancel()

	return msg.Iter(func(i int, p types.Part) error {
		metadata := map[string]*string{}
		p.Metadata().Iter(func(k, v string) error {
			metadata[k] = aws.String(v)
			return nil
		})

		if _, err := a.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
			Bucket:      &a.conf.Bucket,
			Key:         aws.String(a.path.Get(message.Lock(msg, i))),
			Body:        bytes.NewReader(p.Get()),
			ContentType: &a.conf.ContentType,
			Metadata:    metadata,
		}); err != nil {
			return err
		}
		return nil
	})
}

// CloseAsync begins cleaning up resources used by this reader asynchronously.
func (a *AmazonS3) CloseAsync() {
}

// WaitForClose will block until either the reader is closed or a specified
// timeout occurs.
func (a *AmazonS3) WaitForClose(time.Duration) error {
	return nil
}

//------------------------------------------------------------------------------
