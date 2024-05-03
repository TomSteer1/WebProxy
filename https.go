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
	"runtime"
	"strings"
	"time"
)

var generatedHosts = make(map[string]string)

var server *http.Server
var unixListener net.Listener

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

	Debug.Printf("Tunneling request to %s from %s ", originalHost, r.RemoteAddr)
	var destConn net.Conn
	var err error
	if runtime.GOOS == "linux" {
		destConn, err = net.Dial("unix", config.SocketLocation)
	} else {
		destConn, err = net.DialTimeout("tcp", "localhost:8080", 10*time.Second)
	}

	if handleError(err, "Error dialing connection", false) {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		handleError(err, "Error hijacking connection", false)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if handleError(err, "Error hijacking connection", false) {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

func generateSSLHost(host string) {
	// Generate a new SSL certificate for the host
	Debug.Printf("Generating SSL certificate for host %s", host)
	generatedHosts[host] = "localhost"
	// Load ca cert and key
	// Write cert and key to file from embedded resources
	// Create directory for certs
	os.MkdirAll(config.DataDir+"certs", 0755)
	crt, err := secretFs.ReadFile("certs/ca.crt")
	handleError(err, "Error reading ca.crt", false)
	key, err := secretFs.ReadFile("certs/ca.key")
	handleError(err, "Error reading ca.key", false)
	os.WriteFile(config.DataDir+"certs/ca.crt", crt, 0644)
	os.WriteFile(config.DataDir+"certs/ca.key", key, 0644)
	caCert, err := tls.LoadX509KeyPair(config.DataDir+"certs/ca.crt", config.DataDir+"certs/ca.key")
	// caCert, err := tls.LoadX509KeyPair("certs/ca.crt", "certs/ca.key")
	handleError(err, "Error loading CA cert", false)
	// Generate new cert
	cert, key, err := generateCert(caCert)
	handleError(err, "Error generating cert", false)
	// Write cert and key to file
	err = writeCert(cert, key)
	handleError(err, "Error writing cert", false)
	startHttpsServer(config.DataDir+"certs/tempserver.crt", config.DataDir+"certs/tempserver.key")
}

func generateCert(ca tls.Certificate) ([]byte, []byte, error) {
	dnsNames := []string{}
	for k := range generatedHosts {
		dnsNames = append(dnsNames, "*."+k, k)
	}
	// Generate random serial number
	randInt, _ := rand.Int(rand.Reader, big.NewInt(100000))
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
	if handleError(err, "Error generating key", false) {
		return nil, nil, err
	}
	x5ca, _ := x509.ParseCertificate(ca.Certificate[0])
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, x5ca, &priv.PublicKey, ca.PrivateKey)
	if handleError(err, "Error creating cert", false) {
		return nil, nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return certPEM, keyPEM, nil
}

func writeCert(cert []byte, key []byte) error {
	err := os.WriteFile(config.DataDir+"certs/tempserver.crt", cert, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(config.DataDir+"certs/tempserver.key", key, 0644)
	if err != nil {
		return err
	}
	return nil
}

func startSocket() *net.Listener {
	Info.Println("Starting HTTPS socket")
	// If linux, remove socket file
	var unixListener net.Listener
	var err error
	if runtime.GOOS == "linux" {
		Debug.Println("Using unix socket")
		os.Remove(config.SocketLocation)
		os.MkdirAll(config.DataDir, 0755)
		unixListener, err = net.Listen("unix", config.SocketLocation)
	} else {
		unixListener, err = net.Listen("tcp", "localhost:8080")
	}
	handleError(err, "Error starting listener", true)
	return &unixListener

}

func startHttpsServer(certs ...string) *http.Server {
	if server != nil {
		server.Close()
	}
	Info.Println("Starting HTTPS socket")
	server = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Scheme = "https"
			r.URL.Host = r.Host
			handleRequest(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	unixListener = *startSocket()
	if len(certs) == 0 {
		// go server.ListenAndServeTLS(config.SSLCert, config.SSLKey)
		go server.ServeTLS(unixListener, config.SSLCert, config.SSLKey)
	} else {
		// go server.ListenAndServeTLS(certs[0], certs[1])
		go server.ServeTLS(unixListener, certs[0], certs[1])
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
	if handleError(err, "Error in handlePass", false) {
		http.Error(pr.Writer, err.Error(), http.StatusServiceUnavailable)
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
		pr.queueResponse()
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

func loadSSL() {
	os.MkdirAll(config.DataDir+"certs", 0755)
	crt, err := secretFs.ReadFile("certs/server.crt")
	handleError(err, "Error reading from embedded resources", false)
	key, err := secretFs.ReadFile("certs/server.key")
	handleError(err, "Error reading from embedded resources", false)
	os.WriteFile(config.DataDir+"certs/server.crt", crt, 0644)
	os.WriteFile(config.DataDir+"certs/server.key", key, 0644)
}
