package server

import (
	"crypto/ecdsa"
	"encoding/base64"

	"github.com/labstack/echo/v4"
)

type OauthJwksKey struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Use string `json:"use"`
	Alg string `json:"alg"`
}

type OauthJwksResponse struct {
	Keys []OauthJwksKey `json:"keys"`
}

func (s *Server) handleOauthJwks(e echo.Context) error {
	if s.privateKey == nil {
		return e.JSON(200, OauthJwksResponse{Keys: []OauthJwksKey{}})
	}

	pubKey := s.privateKey.Public().(*ecdsa.PublicKey)
	xBytes := padTo32Bytes(pubKey.X.Bytes())
	yBytes := padTo32Bytes(pubKey.Y.Bytes())

	key := OauthJwksKey{
		Kty: "EC",
		Crv: "P-256",
		X:   base64.RawURLEncoding.EncodeToString(xBytes),
		Y:   base64.RawURLEncoding.EncodeToString(yBytes),
		Use: "sig",
		Alg: "ES256",
	}

	return e.JSON(200, OauthJwksResponse{Keys: []OauthJwksKey{key}})
}

func padTo32Bytes(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return padded
}
