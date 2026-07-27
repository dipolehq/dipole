/*
Package awsutil provides shared AWS cryptographic primitives used by both
the identity evidence gatherer (STS GetCallerIdentity signing) and the
credential injection layer (SigV4 request signing).
*/
package awsutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func HmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func DeriveSigningKey(secret, dateStamp, region, service string) []byte {
	kDate := HmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := HmacSHA256(kDate, []byte(region))
	kService := HmacSHA256(kRegion, []byte(service))
	return HmacSHA256(kService, []byte("aws4_request"))
}
