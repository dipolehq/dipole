/*
sigv4.go implements AWS Signature Version 4 request signing for the STS
GetCallerIdentity request. This is a minimal, purpose-built implementation
that signs exactly one request type — no generalization needed.

The canonical request format per AWS spec:

	HTTPMethod\nCanonicalURI\nCanonicalQueryString\nCanonicalHeaders\nSignedHeaders\nHashedPayload

Where CanonicalHeaders is "header:value\n" for each header (already ends with \n).
The \n between CanonicalHeaders and SignedHeaders is the trailing \n of the last header.
*/
package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func signSTSGetCallerIdentity(accessKeyID, secretKey, sessionToken, region string) (map[string]string, error) {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	host := fmt.Sprintf("sts.%s.amazonaws.com", region)
	body := "Action=GetCallerIdentity&Version=2011-06-15"
	contentType := "application/x-www-form-urlencoded; charset=utf-8"

	bodyHash := sha256Hex([]byte(body))

	signedHeaders := "content-type;host;x-amz-date;x-amz-security-token"

	// CanonicalHeaders: each header lowercase, sorted, terminated with \n.
	// The block itself ends with \n (trailing newline of last header line).
	canonicalHeaders := "content-type:" + contentType + "\n" +
		"host:" + host + "\n" +
		"x-amz-date:" + amzDate + "\n" +
		"x-amz-security-token:" + sessionToken + "\n"

	// AWS SigV4 canonical request. Per the spec, the formula is:
	// CanonicalHeaders + '\n' + SignedHeaders + '\n' + HashedPayload
	// Since canonicalHeaders already ends with '\n', this produces a blank
	// line between headers and signed headers — which is correct per AWS docs.
	canonicalRequest := "POST" + "\n" +
		"/" + "\n" +
		"" + "\n" +
		canonicalHeaders + "\n" +
		signedHeaders + "\n" +
		bodyHash

	credentialScope := dateStamp + "/" + region + "/sts/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256" + "\n" +
		amzDate + "\n" +
		credentialScope + "\n" +
		sha256Hex([]byte(canonicalRequest))

	signingKey := deriveSigningKey(secretKey, dateStamp, region, "sts")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authorization := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKeyID, credentialScope, signedHeaders, signature)

	return map[string]string{
		"Authorization":        authorization,
		"Host":                 host,
		"X-Amz-Date":           amzDate,
		"X-Amz-Security-Token": sessionToken,
		"Content-Type":         contentType,
	}, nil
}

func deriveSigningKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
