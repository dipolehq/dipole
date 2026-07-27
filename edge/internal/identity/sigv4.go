/*
sigv4.go implements AWS Signature Version 4 request signing for the STS
GetCallerIdentity request. This is a minimal, purpose-built implementation
that signs exactly one request type — no generalization needed.
*/
package identity

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dipolehq/dipole/edge/internal/awsutil"
)

func signSTSGetCallerIdentity(accessKeyID, secretKey, sessionToken, region string) (map[string]string, error) {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	host := fmt.Sprintf("sts.%s.amazonaws.com", region)
	body := "Action=GetCallerIdentity&Version=2011-06-15"
	contentType := "application/x-www-form-urlencoded; charset=utf-8"

	bodyHash := awsutil.SHA256Hex([]byte(body))

	signedHeaders := "content-type;host;x-amz-date;x-amz-security-token"

	canonicalHeaders := "content-type:" + contentType + "\n" +
		"host:" + host + "\n" +
		"x-amz-date:" + amzDate + "\n" +
		"x-amz-security-token:" + sessionToken + "\n"

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
		awsutil.SHA256Hex([]byte(canonicalRequest))

	signingKey := awsutil.DeriveSigningKey(secretKey, dateStamp, region, "sts")
	signature := hex.EncodeToString(awsutil.HmacSHA256(signingKey, []byte(stringToSign)))

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
