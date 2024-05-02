package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var generatedHosts = make(map[string]string)

var server *http.Server

func handleRequest(w http.ResponseWriter, r *http.Request) {
	var req *ProxyRequest = &ProxyRequest{Request: r, Writer: w, Secure: false, Handled: false, TimeStamp: time.Now().UnixMilli()}
	if r.TLS != nil {
		req.Secure = true
		Debug.Println("Secure request received")

	} else {
		Debug.Println("Insecure request received")
	}
	req.UUID = generateUUID()
	if settings.Enabled && checkRules(req) {
		// Add to queue
		queueRequest(req)

		for !req.Handled && settings.Enabled {
			// Wait until the request is handled
			time.Sleep(1 * time.Second)
		}

		if !req.Handled {
			passUUID(req.UUID)
		}

	} else {
		addToHistory(req)
	}
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	if req.Secure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		tr = &http.Transport{
			DisableCompression: true,
		}
	}
	handlePass(tr, req)
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	// Change host to be a locally controlled host
	var originalHost string = r.Host
	var hostParts = strings.Split(r.URL.Hostname(), ".")
	if len(hostParts) > 2 {
		hostParts = hostParts[1:]
	}
	var baseDomain string = strings.Join(hostParts, ".")
	if _, ok := generatedHosts[baseDomain]; !ok {
		generateSSLHost(baseDomain)
	}

	var newHost string = "localhost:8080"
	Debug.Printf("Tunneling request to %s from %s ", originalHost, r.RemoteAddr)
	dest_conn, err := net.DialTimeout("tcp", newHost, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func generateSSLHost(host string) {
	// Generate a new SSL certificate for the host
	Debug.Printf("Generating SSL certificate for host %s", host)
	generatedHosts[host] = "localhost"
	// Load ca cert and key
	caCert, err := tls.LoadX509KeyPair("certs/ca.crt", "certs/ca.key")
	handleError(err, "Error loading CA cert", false)
	// Generate new cert
	cert, key, err := generateCert(caCert)
	handleError(err, "Error generating cert", false)
	// Write cert and key to file
	err = writeCert(cert, key)
	if err != nil {
		handleError(err, "Error writing cert", false)
	}
	startHttpsServer("certs/tempserver.crt", "certs/tempserver.key")
}

func generateCert(ca tls.Certificate) ([]byte, []byte, error) {
	dnsNames := []string{}
	for k := range generatedHosts {
		dnsNames = append(dnsNames, "*."+k, k)
	}
	// Generate random serial number
	randInt, err := rand.Int(rand.Reader, big.NewInt(100000))
	cert := &x509.Certificate{
		SerialNumber: randInt,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              dnsNames,
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	x5ca, _ := x509.ParseCertificate(ca.Certificate[0])
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, x5ca, &priv.PublicKey, ca.PrivateKey)

	if err != nil {
		return nil, nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return certPEM, keyPEM, nil
}

func writeCert(cert []byte, key []byte) error {
	err := os.WriteFile("certs/tempserver.crt", cert, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile("certs/tempserver.key", key, 0644)
	if err != nil {
		return err
	}
	return nil
}

func startHttpsServer(certs ...string) *http.Server {
	if server != nil {
		server.Close()
	}

	Info.Println("Starting HTTPS server")
	server = &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Scheme = "https"
			r.URL.Host = r.Host
			handleRequest(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	if len(certs) == 0 {
		go server.ListenAndServeTLS(config.SSLCert, config.SSLKey)
	} else {
		go server.ListenAndServeTLS(certs[0], certs[1])
	}
	return server
}

func handlePass(tr *http.Transport, pr *ProxyRequest) {
	// Manually disable compression to avoid issues with decompression
	tr.DisableCompression = true
	if pr.Request.Header.Get("Accept-Encoding") != "" {
		pr.Request.Header.Del("Accept-Encoding")
	}
	resp, err := tr.RoundTrip(pr.Request)
	if err != nil {
		http.Error(pr.Writer, err.Error(), http.StatusServiceUnavailable)
		handleError(err, "Error in handlePass", false)
		return
	}
	// Update history
	pr.Response = resp
	if !settings.CatchResponse || !settings.Enabled || !pr.Handled {
		defer resp.Body.Close()
		copyHeader(pr.Writer.Header(), resp.Header)
		pr.Writer.WriteHeader(resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		pr.Writer.Write(body)
		pr.Response.Body = io.NopCloser(strings.NewReader(string(body)))
		return
	} else {
		defer resp.Body.Close()
		queueResponse(pr)
		for !pr.Handled && settings.Enabled && settings.CatchResponse {
			time.Sleep(1 * time.Second)
		}

		if !pr.Handled {
			passRespUUID(pr.UUID)
		}

		copyHeader(pr.Writer.Header(), resp.Header)
		pr.Writer.WriteHeader(resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		pr.Writer.Write(body)
		pr.Response.Body = io.NopCloser(strings.NewReader(string(body)))
	}
}
