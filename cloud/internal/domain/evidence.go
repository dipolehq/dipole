package domain

type ClientEvidence struct {
	SourceIP       string
	Hostname       string
	K8s            *K8sEvidence
	AWS            *AWSEvidence
	Azure          *AzureEvidence
	GCP            *GCPEvidence
	GitHub         *IdentityTokenEvidence
	GitLab         *IdentityTokenEvidence
	Terraform      *IdentityTokenEvidence
	Process        *ProcessEvidence
	Environment    map[string]string
	DipoleClientID string
	RedirectURI    string
}

type K8sEvidence struct {
	ServiceAccountToken string
	Namespace           string
	PodName             string
	ServiceAccountName  string
	ServiceAccountUID   string
}

type AWSEvidence struct {
	InstanceIdentityDocument  string
	InstanceIdentitySignature string
	STSGetCallerIdentity      *STSGetCallerIdentity
}

type STSGetCallerIdentity struct {
	SignedHeaders map[string]string
	Region        string
}

type AzureEvidence struct {
	Encoding  string
	Signature string
	Nonce     string
}

type GCPEvidence struct {
	IdentityToken string
}

type IdentityTokenEvidence struct {
	IdentityToken string
}

type ProcessEvidence struct {
	Name        string
	Path        string
	CommandLine string
	UserName    string
}

type ServerTarget struct {
	Host string
	Port int
}
