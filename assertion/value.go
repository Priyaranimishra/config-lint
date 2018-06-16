package assertion

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// StandardValueSource can fetch values from external sources
type StandardValueSource struct{}

// GetValue looks up external values when an Expression includes a ValueFrom attribute
func (v StandardValueSource) GetValue(expression Expression) (string, error) {
	if expression.ValueFrom.URL != "" {
		Debugf("Getting value_from %s\n", expression.ValueFrom.URL)
		parsedURL, err := url.Parse(expression.ValueFrom.URL)
		if err != nil {
			return "", err
		}
		switch strings.ToLower(parsedURL.Scheme) {
		case "s3":
			return v.GetValueFromS3(parsedURL.Host, parsedURL.Path)
		case "http":
			return v.GetValueFromHTTP(expression.ValueFrom.URL)
		case "https":
			return v.GetValueFromHTTP(expression.ValueFrom.URL)
		default:
			return "", fmt.Errorf("Unsupported protocol for value_from: %s", parsedURL.Scheme)
		}
	}
	return expression.Value, nil
}

// GetValueFromS3 looks up external values for an Expression when the S3 protocol is specified
func (v StandardValueSource) GetValueFromS3(bucket string, key string) (string, error) {
	region, err := getBucketRegion(bucket)
	if err != nil {
		Debugf("Error getting region for bucket: %s\n", err.Error())
		return "", err
	}

	config := &aws.Config{Region: aws.String(region)}
	awsSession := session.New()
	s3Client := s3.New(awsSession, config)
	response, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		Debugf("Error reading from S3: %s\n", err.Error())
		return "", err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	value := strings.TrimSpace(buf.String())
	Debugf("Value from bucket %s key %s in region %s: %s\n", bucket, key, region, value)
	return value, nil
}

func getBucketRegion(bucket string) (string, error) {
	awsSession := session.New()
	s3Client := s3.New(awsSession)
	location, err := s3Client.GetBucketLocation(&s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		Debugf("Error getting bucket location: %s\n", err.Error())
		return "us-east-1", err
	}
	return *location.LocationConstraint, nil
}

// GetValueFromHTTP looks up external value for an Expression when the HTTP protocol is specified
func (v StandardValueSource) GetValueFromHTTP(url string) (string, error) {
	httpResponse, err := http.Get(url)
	if err != nil {
		return "", err
	}
	if httpResponse.StatusCode != 200 {
		return "", err
	}
	defer httpResponse.Body.Close()
	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
