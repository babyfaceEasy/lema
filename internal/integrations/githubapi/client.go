package githubapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/babyfaceeasy/lema/config"
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
	config     config.Config
}

func NewClient(baseURL string, httpClient HttpClient, logger *zap.Logger, cfg *config.Config) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     logger,
		config:     *cfg,
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
func (c *Client) GetCommitsOLD(repositoryName, ownerName string, since, until *time.Time) ([]CommitResponse, error) {
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

func parseNextLink(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}

	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		sections := strings.Split(part, ";")
		if len(sections) < 2 {
			continue
		}
		urlPart := strings.TrimSpace(sections[0])
		relPart := strings.TrimSpace(sections[1])
		if relPart == `rel="next"` {
			// Trim the angle brackets from the URL.
			if len(urlPart) >= 2 && urlPart[0] == '<' && urlPart[len(urlPart)-1] == '>' {
				return urlPart[1 : len(urlPart)-1]
			}
		}
	}
	return ""
}

// GetCommits calls the commits endpoint and returns all the commits attached to the repositoryName.
// It supports optional "since" and "until" query parameters to filter commits.
func (c *Client) GetCommits(repositoryName, ownerName string, since, until *time.Time) ([]CommitResponse, error) {
	url := fmt.Sprintf("%s/%s/%s/commits", c.baseURL, ownerName, repositoryName)
	c.logger.Debug("Fetching commits", zap.String("url", url))
	var allCommits []CommitResponse

	for {

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set the Authorization header if token is available.
		if c.config.GithubToken != "" {
			req.Header.Set("Authorization", c.config.GithubToken)
		}

		q := req.URL.Query()
		if since != nil {
			q.Set("since", since.UTC().Format(time.RFC3339))
		}
		if until != nil {
			q.Set("until", until.UTC().Format(time.RFC3339))
		}
		// Add page size.
		q.Set("page_size", "100")

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
		// merge them to gether
		allCommits = append(allCommits, commits...)

		linkHeader := resp.Header.Get("Link")
		nextURL := parseNextLink(linkHeader)
		if nextURL == "" {
			c.logger.Info("No more pages of commits")
			break
		}

		c.logger.Info("Fetching next page of commits", zap.String("nextURL", nextURL))
		url = nextURL
	}

	return allCommits, nil
}

func (c *Client) GetRepositoryDetails(repositoryName, ownerName string) (*RepositoryResponse, error) {
	endpoint := fmt.Sprintf(c.baseURL+"/%s/%s", ownerName, repositoryName)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get repository details request: %w", err)
	}

	// Set the Authorization header if token is available.
	if c.config.GithubToken != "" {
		req.Header.Set("Authorization", c.config.GithubToken)
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
