package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type Manager interface {
	Issue(userID, email string) (string, error)
	Validate(raw string) (*Claims, error)
}

type JWT struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
	now      func() time.Time
}

func New(secret, issuer, audience string, ttl time.Duration) (*JWT, error) {
	if len(secret) < 32 || issuer == "" || audience == "" || ttl < time.Second {
		return nil, errors.New("invalid JWT configuration")
	}
	return &JWT{secret: []byte(secret), issuer: issuer, audience: audience, ttl: ttl, now: time.Now}, nil
}

func (j *JWT) Issue(userID, email string) (string, error) {
	if id, err := uuid.Parse(userID); err != nil || id == uuid.Nil || email == "" {
		return "", errors.New("user identity is required")
	}
	now := j.now()
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{j.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
			ID:        uuid.NewString(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
}

func (j *JWT) Validate(raw string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(_ *jwt.Token) (any, error) {
		return j.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(j.issuer), jwt.WithAudience(j.audience), jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(), jwt.WithTimeFunc(j.now))
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if id, err := uuid.Parse(claims.Subject); err != nil || id == uuid.Nil || claims.Email == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
