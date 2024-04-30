#!/bin/bash

# Generate a CA private key
openssl genrsa -out ca.key 2048

# Generate a self-signed CA certificate
openssl req -x509 -new -nodes -key ca.key -sha256 -days 365 -out ca.crt -subj "/CN=2232261"

# Generate a private key for the wildcard domain
openssl genrsa -out wildcard.key 2048

# Generate a certificate signing request (CSR) for the wildcard domain
openssl req -new -key wildcard.key -out wildcard.csr -subj "/CN=duckduckgo.com" 

# Generate a certificate for the wildcard domain signed by the CA
openssl x509 -req -in wildcard.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out wildcard.crt -days 365 -sha256 -extfile <(printf "subjectAltName=DNS:*.*,DNS:duckduckgo.com,DNS:*.com")

# Clean up the CSR and CA serial files
rm wildcard.csr ca.srl