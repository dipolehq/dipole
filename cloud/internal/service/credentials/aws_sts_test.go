// aws_sts_test.go tests AWS STS XML response parsing and invalid-config error handling.
package credentials

import (
	"encoding/xml"
	"testing"
)

const stsAssumeRoleXML = `<AssumeRoleWithWebIdentityResponse>
  <AssumeRoleWithWebIdentityResult>
    <Credentials>
      <AccessKeyId>ASIATESTACCESSKEY</AccessKeyId>
      <SecretAccessKey>testsecretkey123</SecretAccessKey>
      <SessionToken>testsessiontoken456</SessionToken>
      <Expiration>2026-06-26T12:00:00Z</Expiration>
    </Credentials>
  </AssumeRoleWithWebIdentityResult>
</AssumeRoleWithWebIdentityResponse>`

func TestAWSSts_XMLParsing(t *testing.T) {
	var resp assumeRoleResponse
	if err := xml.Unmarshal([]byte(stsAssumeRoleXML), &resp); err != nil {
		t.Fatal(err)
	}

	creds := resp.Result.Credentials
	if creds.AccessKeyID != "ASIATESTACCESSKEY" {
		t.Errorf("AccessKeyId = %s, want ASIATESTACCESSKEY", creds.AccessKeyID)
	}
	if creds.SecretAccessKey != "testsecretkey123" {
		t.Errorf("SecretAccessKey = %s", creds.SecretAccessKey)
	}
	if creds.SessionToken != "testsessiontoken456" {
		t.Errorf("SessionToken = %s", creds.SessionToken)
	}
	if creds.Expiration != "2026-06-26T12:00:00Z" {
		t.Errorf("Expiration = %s", creds.Expiration)
	}
}

func TestAWSSts_InvalidConfig_Error(t *testing.T) {
	handler := NewAWSStsHandler(nil)
	_, err := handler.Retrieve(nil, []byte("not json"), "https://647251-id.dipolehq.com", nil, nil)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}
