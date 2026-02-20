package models

import (
    "crypto/rand"
    "database/sql"
    "encoding/hex"
    "fmt"
    "math/big"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
    "golang.org/x/crypto/sha3"
)

const (
    bcryptCost      = 10
    shake256Length   = 16 // bytes → 32 hex chars
    jwtExpiration   = 30 * 24 * time.Hour
    secretAlphabet  = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    secretLength    = 64
)

type User struct {
    ID       int
    Username string
    Password string // bcrypt hash
    Active   bool
}

type JWTClaims struct {
    Username string `json:"username"`
    H        string `json:"h"`
    jwt.RegisteredClaims
}

type UserStore struct {
    db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
    return &UserStore{db: db}
}

// FindByUsername returns the user or nil if not found.
func (s *UserStore) FindByUsername(username string) (*User, error) {
    u := &User{}
    err := s.db.QueryRow(
        "SELECT id, username, password, active FROM user WHERE username = ? AND active = 1",
        username,
    ).Scan(&u.ID, &u.Username, &u.Password, &u.Active)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("find user: %w", err)
    }
    return u, nil
}

// Count returns the number of users in the database.
func (s *UserStore) Count() (int, error) {
    var count int
    err := s.db.QueryRow("SELECT COUNT(*) FROM user").Scan(&count)
    return count, err
}

// Create inserts a new user with a bcrypt-hashed password.
func (s *UserStore) Create(username, password string) (*User, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    if err != nil {
        return nil, fmt.Errorf("hash password: %w", err)
    }

    res, err := s.db.Exec(
        "INSERT INTO user (username, password, active) VALUES (?, ?, 1)",
        username, string(hash),
    )
    if err != nil {
        return nil, fmt.Errorf("insert user: %w", err)
    }

    id, _ := res.LastInsertId()
    return &User{
        ID:       int(id),
        Username: username,
        Password: string(hash),
        Active:   true,
    }, nil
}

// ChangePassword updates the user's password.
func (s *UserStore) ChangePassword(userID int, newPassword string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
    if err != nil {
        return fmt.Errorf("hash password: %w", err)
    }
    _, err = s.db.Exec("UPDATE user SET password = ? WHERE id = ?", string(hash), userID)
    return err
}

// VerifyPassword checks a plaintext password against the stored bcrypt hash.
func VerifyPassword(password, hash string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CreateJWT creates an HS256 JWT token for the user.
func CreateJWT(user *User, secret string) (string, error) {
    claims := JWTClaims{
        Username: user.Username,
        H:        Shake256Hex(user.Password, shake256Length),
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtExpiration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

// VerifyJWT parses and validates a JWT token.
// Returns claims or error. Accepts tokens without exp (Node.js compat).
func VerifyJWT(tokenString, secret string) (*JWTClaims, error) {
    parser := jwt.NewParser(
        jwt.WithValidMethods([]string{"HS256"}),
        // Don't require exp — Node.js tokens don't have it
        jwt.WithExpirationRequired(),
    )

    // Try with exp required first; if that fails due to missing exp, retry without
    token, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })
    if err != nil {
        // Retry without exp requirement for Node.js token compat
        parser = jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))
        token, err = parser.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
            return []byte(secret), nil
        })
        if err != nil {
            return nil, fmt.Errorf("invalid token: %w", err)
        }
    }

    claims, ok := token.Claims.(*JWTClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token claims")
    }
    return claims, nil
}

// Shake256Hex computes SHAKE256 of data and returns the first `length` bytes as hex.
func Shake256Hex(data string, length int) string {
    if data == "" {
        return ""
    }
    h := sha3.NewShake256()
    h.Write([]byte(data))
    out := make([]byte, length)
    h.Read(out)
    return hex.EncodeToString(out)
}

// GenSecret generates a cryptographically random alphanumeric string.
func GenSecret(length int) (string, error) {
    b := make([]byte, length)
    for i := range b {
        n, err := rand.Int(rand.Reader, big.NewInt(int64(len(secretAlphabet))))
        if err != nil {
            return "", err
        }
        b[i] = secretAlphabet[n.Int64()]
    }
    return string(b), nil
}
