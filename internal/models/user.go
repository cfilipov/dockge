package models

import (
    "crypto/rand"
    "encoding/binary"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "math/big"
    "time"

    bolt "go.etcd.io/bbolt"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
    "golang.org/x/crypto/sha3"

    "github.com/cfilipov/dockge/internal/db"
)

const (
    bcryptCost     = 10
    shake256Length  = 16 // bytes → 32 hex chars
    jwtExpiration  = 30 * 24 * time.Hour
    secretAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    secretLength   = 64
)

type User struct {
    ID       int    `json:"id"`
    Username string `json:"username"`
    Password string `json:"password"`
    Active   bool   `json:"active"`
}

type JWTClaims struct {
    Username string `json:"username"`
    H        string `json:"h"`
    jwt.RegisteredClaims
}

type UserStore struct {
    db *bolt.DB
}

func NewUserStore(database *bolt.DB) *UserStore {
    return &UserStore{db: database}
}

// itob converts a uint64 to an 8-byte big-endian slice for use as a bbolt key.
func itob(v uint64) []byte {
    b := make([]byte, 8)
    binary.BigEndian.PutUint64(b, v)
    return b
}

// FindByUsername returns the user or nil if not found.
func (s *UserStore) FindByUsername(username string) (*User, error) {
    var u *User
    err := s.db.View(func(tx *bolt.Tx) error {
        v := tx.Bucket(db.BucketUsers).Get([]byte(username))
        if v == nil {
            return nil
        }
        u = &User{}
        if err := json.Unmarshal(v, u); err != nil {
            return fmt.Errorf("unmarshal user: %w", err)
        }
        if !u.Active {
            u = nil
        }
        return nil
    })
    if err != nil {
        return nil, fmt.Errorf("find user: %w", err)
    }
    return u, nil
}

// FindByID returns the user or nil if not found.
func (s *UserStore) FindByID(id int) (*User, error) {
    var u *User
    err := s.db.View(func(tx *bolt.Tx) error {
        // Look up username from ID index
        idKey := itob(uint64(id))
        username := tx.Bucket(db.BucketUsersByID).Get(idKey)
        if username == nil {
            return nil
        }
        // Look up user by username
        v := tx.Bucket(db.BucketUsers).Get(username)
        if v == nil {
            return nil
        }
        u = &User{}
        return json.Unmarshal(v, u)
    })
    if err != nil {
        return nil, fmt.Errorf("find user by id: %w", err)
    }
    return u, nil
}

// Count returns the number of users in the database.
func (s *UserStore) Count() (int, error) {
    var count int
    err := s.db.View(func(tx *bolt.Tx) error {
        count = tx.Bucket(db.BucketUsers).Stats().KeyN
        return nil
    })
    return count, err
}

// Create inserts a new user with a bcrypt-hashed password.
func (s *UserStore) Create(username, password string) (*User, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    if err != nil {
        return nil, fmt.Errorf("hash password: %w", err)
    }

    var u *User
    err = s.db.Update(func(tx *bolt.Tx) error {
        // Get next ID from the users_by_id bucket sequence
        idBucket := tx.Bucket(db.BucketUsersByID)
        seq, err := idBucket.NextSequence()
        if err != nil {
            return fmt.Errorf("next sequence: %w", err)
        }

        u = &User{
            ID:       int(seq),
            Username: username,
            Password: string(hash),
            Active:   true,
        }

        data, err := json.Marshal(u)
        if err != nil {
            return fmt.Errorf("marshal user: %w", err)
        }

        // Store user by username
        if err := tx.Bucket(db.BucketUsers).Put([]byte(username), data); err != nil {
            return err
        }

        // Store ID → username index
        return idBucket.Put(itob(seq), []byte(username))
    })
    if err != nil {
        return nil, fmt.Errorf("create user: %w", err)
    }
    return u, nil
}

// ChangePassword updates the user's password.
func (s *UserStore) ChangePassword(userID int, newPassword string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
    if err != nil {
        return fmt.Errorf("hash password: %w", err)
    }

    return s.db.Update(func(tx *bolt.Tx) error {
        // Look up username from ID
        idKey := itob(uint64(userID))
        username := tx.Bucket(db.BucketUsersByID).Get(idKey)
        if username == nil {
            return fmt.Errorf("user id %d not found", userID)
        }

        bucket := tx.Bucket(db.BucketUsers)
        v := bucket.Get(username)
        if v == nil {
            return fmt.Errorf("user %q not found", string(username))
        }

        var u User
        if err := json.Unmarshal(v, &u); err != nil {
            return fmt.Errorf("unmarshal user: %w", err)
        }

        u.Password = string(hash)

        data, err := json.Marshal(&u)
        if err != nil {
            return fmt.Errorf("marshal user: %w", err)
        }
        return bucket.Put(username, data)
    })
}

// VerifyPassword checks a plaintext password against the stored bcrypt hash.
func VerifyPassword(password, hash string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CreateJWT creates an HS256 JWT token for the user.
func CreateJWT(user *User, secret string) (string, error) {
    now := time.Now()
    claims := JWTClaims{
        Username: user.Username,
        H:        Shake256Hex(user.Password, shake256Length),
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(now.Add(jwtExpiration)),
            IssuedAt:  jwt.NewNumericDate(now),
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
