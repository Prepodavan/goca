package models

type Payload struct {
	Body []byte
}

type CSR Payload
type OpenSSLConfig Payload

type File struct {
	Payload
	Form string
}

type Key File
type Certificate File

type Produced struct {
	Certificate     Certificate
	RootCertificate Certificate
	Request         CSR
	PrivateKey      *Key
	PublicKey       Key
	Config          *OpenSSLConfig
}
