package models

import (
    "database/sql"
    "fmt"
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
    db *sql.DB
}

func NewAgentStore(db *sql.DB) *AgentStore {
    return &AgentStore{db: db}
}

// GetAll returns all agents.
func (s *AgentStore) GetAll() ([]Agent, error) {
    rows, err := s.db.Query("SELECT id, url, username, password, name, active FROM agent")
    if err != nil {
        return nil, fmt.Errorf("get agents: %w", err)
    }
    defer rows.Close()

    var agents []Agent
    for rows.Next() {
        var a Agent
        if err := rows.Scan(&a.ID, &a.URL, &a.Username, &a.Password, &a.Name, &a.Active); err != nil {
            return nil, err
        }
        agents = append(agents, a)
    }
    return agents, rows.Err()
}

// Add inserts a new agent.
func (s *AgentStore) Add(url, username, password, name string) (*Agent, error) {
    res, err := s.db.Exec(
        "INSERT INTO agent (url, username, password, name, active) VALUES (?, ?, ?, ?, 1)",
        url, username, password, name,
    )
    if err != nil {
        return nil, fmt.Errorf("add agent: %w", err)
    }
    id, err := res.LastInsertId()
    if err != nil {
        return nil, fmt.Errorf("get agent id: %w", err)
    }
    return &Agent{
        ID:       int(id),
        URL:      url,
        Username: username,
        Password: password,
        Name:     name,
        Active:   true,
    }, nil
}

// Remove deletes an agent by URL.
func (s *AgentStore) Remove(url string) error {
    _, err := s.db.Exec("DELETE FROM agent WHERE url = ?", url)
    return err
}

// UpdateName changes an agent's display name.
func (s *AgentStore) UpdateName(url, name string) error {
    _, err := s.db.Exec("UPDATE agent SET name = ? WHERE url = ?", name, url)
    return err
}
