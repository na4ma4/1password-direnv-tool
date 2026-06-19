package model

type ExecCredential struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Status     *ExecCredentialStatus `json:"status,omitempty"`
	Spec       *ExecCredentialSpec   `json:"spec,omitempty"`
}

type ExecCredentialStatus struct {
	ClientCertificateData string `json:"clientCertificateData,omitempty"`
	ClientKeyData         string `json:"clientKeyData,omitempty"`
	ExpirationTimestamp   string `json:"expirationTimestamp,omitempty"`
}

type ExecCredentialSpec struct {
	Cluster     *ExecCredentialClusterInfo `json:"cluster,omitempty"`
	Interactive bool                       `json:"interactive,omitempty"`
}

type ExecCredentialClusterInfo struct {
	Server                   string `json:"server,omitempty"`
	CertificateAuthorityData string `json:"certificate-authority-data,omitempty"`
	// Config                   any    `json:"config,omitempty"`
}

func NewExecCredential() *ExecCredential {
	return &ExecCredential{
		APIVersion: "client.authentication.k8s.io/v1",
		Kind:       "ExecCredential",
		Status:     &ExecCredentialStatus{},
		Spec: &ExecCredentialSpec{
			Cluster: &ExecCredentialClusterInfo{},
		},
	}
}
