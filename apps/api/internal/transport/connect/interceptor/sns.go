package interceptor

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"connectrpc.com/connect"
)

// SNS certificate cache to avoid re-fetching
var (
	certCache   = make(map[string]*x509.Certificate)
	certCacheMu sync.RWMutex
)

// NewSNSInterceptor creates an interceptor that validates SNS message signatures.
// For now, this is a placeholder - full SNS signature verification can be added later.
func NewSNSInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure

			// Only check for SnsService endpoints
			if !strings.HasPrefix(procedure, "/emailapi.v1.SnsService/") {
				return next(ctx, req)
			}

			// Check for SNS message type header
			messageType := req.Header().Get("X-Amz-Sns-Message-Type")
			if messageType == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing X-Amz-Sns-Message-Type header"))
			}

			// For now, just validate the header is present
			// Full signature verification would decode the message and verify against the signing certificate

			return next(ctx, req)
		}
	}
}

// fetchCertificate fetches and caches an X.509 certificate from a URL.
// This is used for SNS signature verification.
func fetchCertificate(certURL string) (*x509.Certificate, error) {
	// Check cache first
	certCacheMu.RLock()
	cert, ok := certCache[certURL]
	certCacheMu.RUnlock()
	if ok {
		return cert, nil
	}

	// Fetch certificate
	resp, err := http.Get(certURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse PEM
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, errors.New("failed to parse certificate PEM")
	}

	cert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	// Cache the certificate
	certCacheMu.Lock()
	certCache[certURL] = cert
	certCacheMu.Unlock()

	return cert, nil
}

// verifySignature verifies an SNS message signature.
// This is a helper for full SNS verification if needed.
func verifySignature(cert *x509.Certificate, message, signature string) error {
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}

	return cert.CheckSignature(x509.SHA1WithRSA, []byte(message), sig)
}
