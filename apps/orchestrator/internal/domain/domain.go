package domain

import (
	"fmt"
	"time"
)

// Domain represents a domain that can be connected to projects
type Domain struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProjectDomain represents a domain connected to a project with a prefix
type ProjectDomain struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	DomainID  int64     `json:"domain_id"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"created_at"`
}

// ProjectDomainDetail includes domain name for display
type ProjectDomainDetail struct {
	ID         int64  `json:"id"`
	ProjectID  int64  `json:"project_id"`
	DomainID   int64  `json:"domain_id"`
	DomainName string `json:"domain_name"`
	Prefix     string `json:"prefix"`
}

// ConnectProjectRequest represents the request to connect a domain to a project
type ConnectProjectRequest struct {
	ProjectID int64  `json:"project_id"`
	Prefix    string `json:"prefix"`
}

// GeneratePreviewURL generates a preview URL for a deployment
// Format: {prefix}-{short_commit_sha}.{domain}
// Example: preview-a1b2c3d.example.com
func GeneratePreviewURL(prefix, domainName, commitSHA string) string {
	shortSHA := commitSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}
	return fmt.Sprintf("%s-%s.%s", prefix, shortSHA, domainName)
}
