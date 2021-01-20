package grafana

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Organization maps to a Grafana.com organization.
type Organization struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

// ErrOrganizationNotFound is returned when we couldn't find an organization with a given slug.
var ErrOrganizationNotFound = errors.New("organization not found")

// ErrPrivateOrganization is returned when an organization exists but hasn't published any plugins yet.
var ErrPrivateOrganization = errors.New("organization is private")

// Client provides operations to the grafana.com API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a new Client.
func NewClient() *Client {
	return &Client{
		baseURL:    "https://grafana.com/api",
		httpClient: &http.Client{},
	}
}

// FindOrgBySlug returns the organization with a given slug.
func (c *Client) FindOrgBySlug(slug string) (*Organization, error) {
	ok, err := c.usernameExists(slug)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrOrganizationNotFound
	}

	req, err := http.NewRequest("GET", c.baseURL+"/orgs/"+slug, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrPrivateOrganization
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var org Organization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, err
	}

	return &org, nil
}

// usernameExists checks whether a username is available on Grafana.com.
func (c *Client) usernameExists(username string) (bool, error) {
	body := strings.NewReader(fmt.Sprintf(`{"slug": "%s"}`, username))

	req, err := http.NewRequest("POST", c.baseURL+"/orgs/check-slug", body)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// 200 means the username is available.
	if resp.StatusCode != http.StatusOK {
		// 409 means it's already taken.
		if resp.StatusCode == http.StatusConflict {
			return true, nil
		}
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return false, nil
}

// PluginVersion contains information about a published plugin.
type PluginVersion struct {
	Version   string    `json:"version"`
	Commit    string    `json:"commit"`
	CreatedAt time.Time `json:"createdAt"`
	Downloads int       `json:"downloads"`
	URL       string    `json:"url"`
	Verified  bool      `json:"verified"`
}

// FindPluginVersions returns all published versions for a given plugin ID.
func (c *Client) FindPluginVersions(pluginID string) ([]PluginVersion, error) {
	var data struct {
		Items []PluginVersion `json:"items"`
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/plugins/%s/versions", c.baseURL, pluginID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data.Items, nil
}
