package githubapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseURL    string
	httpClient HttpClient
	logger     *zap.Logger
}

func NewClient(baseURL string, httpClient HttpClient, logger *zap.Logger) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     logger,
	}
}

// GitHubError represents an error response from the GitHub API.
type GitHubError struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url"`
	Status           string `json:"status"`
}

// Person represents details for the commit's author or committer.
type Person struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"` // Expected format: RFC3339
}

// CommitDetail holds detailed information about the commit.
type CommitDetail struct {
	Author       Person `json:"author"`
	Committer    Person `json:"committer"`
	Message      string `json:"message"`
	URL          string `json:"url"`
	CommentCount int    `json:"comment_count"`
}

type CommitResponse struct {
	SHA    string       `json:"sha"`
	URL    string       `json:"url"`
	Commit CommitDetail `json:"commit"`
}

type RepositoryResponse struct {
	Name                string `json:"name"`
	URL                 string `json:"url"`
	Description         string `json:"description"`
	ProgrammingLanguage string `json:"language"`
	ForksCount          int    `json:"forks_count"`
	OpenIssuesCount     int    `json:"open_issues_count"`
	WatchersCount       int    `json:"watchers"`
	StarsCount          int    `json:"stargazers_count"`
}

// GetCommits calls the commits endpoint and returns all the commits attached to the repositoryName.
// It supports optional "since" and "until" query parameters to filter commits.
func (c *Client) GetCommits(repositoryName, ownerName string, since, until *time.Time) ([]CommitResponse, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/commits", c.baseURL, ownerName, repositoryName)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get commits request: %w", err)
	}

	q := req.URL.Query()
	if since != nil {
		q.Set("since", since.UTC().Format(time.RFC3339))
	}
	if until != nil {
		q.Set("until", until.UTC().Format(time.RFC3339))
	}
	req.URL.RawQuery = q.Encode()

	// Execute the request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit get commits http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("error reading response body", zap.Error(err))
	}

	if resp.StatusCode != http.StatusOK {
		var ghErr GitHubError
		if err := sonic.Unmarshal(body, &ghErr); err != nil {
			c.logger.Error("Unexpected status code", zap.Int("status code", resp.StatusCode), zap.Error(err))
		}
		return nil, errors.New(ghErr.Message)
	}

	var commits []CommitResponse
	if err := sonic.Unmarshal(body, &commits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commits http response: %w", err)
	}

	return commits, nil
}

func (c *Client) GetRepositoryDetails(repositoryName, ownerName string) (*RepositoryResponse, error) {
	endpoint := fmt.Sprintf(c.baseURL+"/%s/%s", ownerName, repositoryName)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get repository details request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit get repository details http request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("error reading response body", zap.Error(err))
	}

	if resp.StatusCode != http.StatusOK {
		var ghErr GitHubError
		if err := sonic.Unmarshal(body, &ghErr); err != nil {
			c.logger.Error("Unexpected status code", zap.Int("status code", resp.StatusCode), zap.Error(err))
		}

		return nil, errors.New(ghErr.Message)
	}

	var repo RepositoryResponse
	if err := sonic.Unmarshal(body, &repo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repository http response: %w", err)
	}

	return &repo, nil
}
