package handlers

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"clinic-backend/models"
	"clinic-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const staffContextKey = "staff"
const staffRoleContextKey = "staff_role"

type StaffRole string

const (
	RoleAdmin StaffRole = "admin"
	RoleStaff StaffRole = "staff"
)

// contextStaff extracts the ClinicStaff from the Gin context.
func contextStaff(c *gin.Context) models.ClinicStaff {
	v, _ := c.Get(staffContextKey)
	return v.(models.ClinicStaff)
}

// contextRole extracts the StaffRole from the Gin context.
func contextRole(c *gin.Context) StaffRole {
	v, _ := c.Get(staffRoleContextKey)
	return v.(StaffRole)
}

// jwksKey is a single JWK entry for an RSA public key.
type jwksKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
}

// jwksSet is the JWKS response body.
type jwksSet struct {
	Keys []jwksKey `json:"keys"`
}

// cachedJWKS holds a parsed JWKS with its fetch time.
type cachedJWKS struct {
	keys    map[string]*rsa.PublicKey
	fetched time.Time
	ttl     time.Duration
}

// keycloakMiddleware implements JWT validation using Keycloak JWKS.
type keycloakMiddleware struct {
	realmURL string
	clientID string
	staffSvc *services.StaffService

	mu     sync.RWMutex
	cached *cachedJWKS
}

// NewKeycloakAuthMiddleware creates a Gin middleware that validates
// Bearer JWTs from Keycloak and populates the staff context.
//
// If realmURL is empty, the middleware enforces nothing (for development).
func NewKeycloakAuthMiddleware(realmURL, clientID string, staffSvc *services.StaffService) gin.HandlerFunc {
	m := &keycloakMiddleware{
		realmURL: strings.TrimRight(realmURL, "/"),
		clientID: clientID,
		staffSvc: staffSvc,
	}
	return m.handle
}

func (m *keycloakMiddleware) handle(c *gin.Context) {
	if m.realmURL == "" || m.clientID == "" {
		c.Next()
		return
	}

	tokenStr, err := extractBearer(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed authorization header"})
		return
	}

	token, err := m.parseJWT(tokenStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	groups := extractGroups(claims)
	role := determineRole(groups)
	if role == "" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient group membership"})
		return
	}

	accountID := extractAccountID(claims)
	if accountID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token missing user identifier"})
		return
	}

	realname := extractRealname(claims)
	staff, err := m.staffSvc.GetOrCreateByAccountID(accountID, realname)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve staff"})
		return
	}

	c.Set(staffContextKey, staff)
	c.Set(staffRoleContextKey, role)
	c.Next()
}

func extractBearer(c *gin.Context) (string, error) {
	auth := c.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", errors.New("no bearer token")
	}
	return strings.TrimPrefix(auth, "Bearer "), nil
}

func extractGroups(claims jwt.MapClaims) []string {
	switch v := claims["groups"].(type) {
	case []any:
		groups := make([]string, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
		return groups
	case []string:
		return v
	}
	return nil
}

func extractAccountID(claims jwt.MapClaims) string {
	if v, ok := claims["preferred_username"].(string); ok && v != "" {
		return v
	}
	if v, ok := claims["sub"].(string); ok && v != "" {
		return v
	}
	return ""
}

func extractRealname(claims jwt.MapClaims) string {
	if v, ok := claims["name"].(string); ok && v != "" {
		return v
	}
	return ""
}

func determineRole(groups []string) StaffRole {
	hasManagement := false
	hasClinic := false
	for _, g := range groups {
		lg := strings.ToLower(g)
		if strings.Contains(lg, "management") {
			hasManagement = true
		}
		if strings.Contains(lg, "clinic") {
			hasClinic = true
		}
	}
	if hasManagement && hasClinic {
		return RoleAdmin
	}
	if hasClinic {
		return RoleStaff
	}
	return ""
}

func (m *keycloakMiddleware) parseJWT(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, m.keyFunc,
		jwt.WithIssuer(m.realmURL+"/"),
		jwt.WithAudience(m.clientID),
		jwt.WithValidMethods([]string{"RS256"}),
	)
}

func (m *keycloakMiddleware) keyFunc(token *jwt.Token) (any, error) {
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("token missing kid header")
	}

	keys, err := m.getKeys()
	if err != nil {
		return nil, fmt.Errorf("get jwks: %w", err)
	}

	pub, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("unknown kid %s", kid)
	}
	return pub, nil
}

func (m *keycloakMiddleware) getKeys() (map[string]*rsa.PublicKey, error) {
	m.mu.RLock()
	c := m.cached
	m.mu.RUnlock()

	if c != nil && time.Since(c.fetched) < c.ttl {
		return c.keys, nil
	}

	return m.fetchKeys()
}

func (m *keycloakMiddleware) fetchKeys() (map[string]*rsa.PublicKey, error) {
	url := m.realmURL + "/protocol/openid-connect/certs"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	var set jwksSet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := k.rsaPublicKey()
		if err != nil {
			return nil, fmt.Errorf("parse jwk kid=%s: %w", k.Kid, err)
		}
		keys[k.Kid] = pub
	}

	m.mu.Lock()
	m.cached = &cachedJWKS{
		keys:    keys,
		fetched: time.Now(),
		ttl:     1 * time.Hour,
	}
	m.mu.Unlock()

	return keys, nil
}

func (k jwksKey) rsaPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	if len(eBytes) < 8 {
		pad := make([]byte, 8-len(eBytes))
		eBytes = append(pad, eBytes...)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Uint64())

	return &rsa.PublicKey{N: n, E: e}, nil
}

// SkipKeycloakAuth is a no-op middleware for development.
func SkipKeycloakAuth(c *gin.Context) {
	c.Next()
}

// RequireStaff aborts with 403 if the logged-in staff does not have at least staff role.
func RequireStaff(c *gin.Context) {
	role := contextRole(c)
	if role == "" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.Next()
}

// RequireAdmin aborts with 403 if the logged-in staff is not an admin.
func RequireAdmin(c *gin.Context) {
	if contextRole(c) != RoleAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.Next()
}
