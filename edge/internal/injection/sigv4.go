/*
sigv4.go computes AWS Signature Version 4 for arbitrary HTTP requests.
Used when credential type is aws_sts and target is *.amazonaws.com.
Signs the full request (method, URI, query, headers, body hash) and sets
Authorization, X-Amz-Date, X-Amz-Security-Token, and X-Amz-Content-Sha256.
*/
package injection

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/dipolehq/dipole/edge/internal/awsutil"
)

type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

func SignAWSRequest(req *http.Request, creds *AWSCredentials, region, service string) error {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	bodyHash := "UNSIGNED-PAYLOAD"
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		bodyHash = awsutil.SHA256Hex(bodyBytes)
		req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
	} else {
		bodyHash = awsutil.SHA256Hex([]byte(""))
	}

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", bodyHash)
	if creds.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", creds.SessionToken)
	}
	if req.Header.Get("Host") == "" {
		req.Header.Set("Host", req.Host)
	}

	canonicalURI := req.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalQueryString := canonicalQuery(req.URL.Query())
	canonicalHeaders, signedHeaders := buildCanonicalHeaders(req.Header)

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		bodyHash,
	}, "\n")

	credentialScope := dateStamp + "/" + region + "/" + service + "/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + credentialScope + "\n" + awsutil.SHA256Hex([]byte(canonicalRequest))

	signingKey := awsutil.DeriveSigningKey(creds.SecretAccessKey, dateStamp, region, service)
	signature := hex.EncodeToString(awsutil.HmacSHA256(signingKey, []byte(stringToSign)))

	authorization := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		creds.AccessKeyID, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authorization)

	return nil
}

func ExtractServiceRegion(host string) (service, region string, err error) {
	host = strings.TrimSuffix(host, ".amazonaws.com")
	parts := strings.Split(host, ".")

	switch len(parts) {
	case 1:
		return parts[0], "us-east-1", nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return parts[0], parts[1], nil
	}
}

func canonicalQuery(params url.Values) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		for _, v := range params[k] {
			pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(pairs, "&")
}

func buildCanonicalHeaders(headers http.Header) (canonicalHeaders, signedHeaders string) {
	type headerEntry struct {
		name  string
		value string
	}

	var entries []headerEntry
	for name, values := range headers {
		lowerName := strings.ToLower(name)
		if !shouldSignHeader(lowerName) {
			continue
		}
		trimmedValue := strings.TrimSpace(strings.Join(values, ","))
		entries = append(entries, headerEntry{lowerName, trimmedValue})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	var headerLines []string
	var headerNames []string
	for _, e := range entries {
		headerLines = append(headerLines, e.name+":"+e.value+"\n")
		headerNames = append(headerNames, e.name)
	}

	return strings.Join(headerLines, ""), strings.Join(headerNames, ";")
}

func shouldSignHeader(name string) bool {
	switch {
	case name == "host":
		return true
	case name == "content-type":
		return true
	case strings.HasPrefix(name, "x-amz-"):
		return true
	default:
		return false
	}
}
