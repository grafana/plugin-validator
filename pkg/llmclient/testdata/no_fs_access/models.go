package main

import (
	"fmt"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []string  `json:"tags"`
}

type Plugin struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      User              `json:"author"`
	Metadata    map[string]string `json:"metadata"`
}

func (u User) String() string {
	return fmt.Sprintf("User{id=%s, name=%s, email=%s}", u.ID, u.Name, u.Email)
}

func (p Plugin) String() string {
	return fmt.Sprintf("Plugin{id=%s, name=%s, version=%s}", p.ID, p.Name, p.Version)
}

func (p Plugin) HasMetadata(key string) bool {
	_, ok := p.Metadata[key]
	return ok
}

func NewUser(name, email string) User {
	return User{
		ID:        generateID(),
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
	}
}

func NewPlugin(name, version string, author User) Plugin {
	return Plugin{
		ID:       generateID(),
		Name:     name,
		Version:  version,
		Author:   author,
		Metadata: make(map[string]string),
	}
}
