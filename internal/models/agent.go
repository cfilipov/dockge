package models

import (
    "encoding/json"
    "fmt"

    bolt "go.etcd.io/bbolt"

    "github.com/cfilipov/dockge/internal/db"
)

type Agent struct {
    ID       int    `json:"id"`
    URL      string `json:"url"`
    Username string `json:"username"`
    Password string `json:"password"`
    Name     string `json:"name"`
    Active   bool   `json:"active"`
}

type AgentStore struct {
    db *bolt.DB
}

func NewAgentStore(database *bolt.DB) *AgentStore {
    return &AgentStore{db: database}
}

// GetAll returns all agents.
func (s *AgentStore) GetAll() ([]Agent, error) {
    var agents []Agent
    err := s.db.View(func(tx *bolt.Tx) error {
        return tx.Bucket(db.BucketAgents).ForEach(func(k, v []byte) error {
            var a Agent
            if err := json.Unmarshal(v, &a); err != nil {
                return fmt.Errorf("unmarshal agent %q: %w", string(k), err)
            }
            agents = append(agents, a)
            return nil
        })
    })
    if err != nil {
        return nil, fmt.Errorf("get agents: %w", err)
    }
    return agents, nil
}

// Add inserts a new agent.
func (s *AgentStore) Add(url, username, password, name string) (*Agent, error) {
    a := &Agent{
        URL:      url,
        Username: username,
        Password: password,
        Name:     name,
        Active:   true,
    }

    err := s.db.Update(func(tx *bolt.Tx) error {
        bucket := tx.Bucket(db.BucketAgents)
        // Use bucket sequence for auto-increment ID
        seq, err := bucket.NextSequence()
        if err != nil {
            return fmt.Errorf("next sequence: %w", err)
        }
        a.ID = int(seq)

        data, err := json.Marshal(a)
        if err != nil {
            return fmt.Errorf("marshal agent: %w", err)
        }
        return bucket.Put([]byte(url), data)
    })
    if err != nil {
        return nil, fmt.Errorf("add agent: %w", err)
    }
    return a, nil
}

// Remove deletes an agent by URL.
func (s *AgentStore) Remove(url string) error {
    return s.db.Update(func(tx *bolt.Tx) error {
        return tx.Bucket(db.BucketAgents).Delete([]byte(url))
    })
}

// UpdateName changes an agent's display name.
func (s *AgentStore) UpdateName(url, name string) error {
    return s.db.Update(func(tx *bolt.Tx) error {
        bucket := tx.Bucket(db.BucketAgents)
        v := bucket.Get([]byte(url))
        if v == nil {
            return fmt.Errorf("agent %q not found", url)
        }

        var a Agent
        if err := json.Unmarshal(v, &a); err != nil {
            return fmt.Errorf("unmarshal agent: %w", err)
        }

        a.Name = name

        data, err := json.Marshal(&a)
        if err != nil {
            return fmt.Errorf("marshal agent: %w", err)
        }
        return bucket.Put([]byte(url), data)
    })
}
