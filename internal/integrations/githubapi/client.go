package githubapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

type RepositoryOwner struct {
	Login string `json:"login"`
}

type RepositoryResponse struct {
	Name                string          `json:"name"`
	Owner               RepositoryOwner `json:"owner"`
	URL                 string          `json:"url"`
	Description         string          `json:"description"`
	ProgrammingLanguage string          `json:"language"`
	ForksCount          int             `json:"forks_count"`
	OpenIssuesCount     int             `json:"open_issues_count"`
	WatchersCount       int             `json:"watchers"`
	StarsCount          int             `json:"stargazers_count"`
}

func parseLastPage(linkHeader string) int {
	// Example Link header:
	// <https://api.github.com/repositories/120360765/commits?page=2>; rel="next", <https://api.github.com/repositories/120360765/commits?page=51757>; rel="last"
	if linkHeader == "" {
		return 1
	}
	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		sections := strings.Split(part, ";")
		if len(sections) < 2 {
			continue
		}
		urlPart := strings.TrimSpace(sections[0])
		relPart := strings.TrimSpace(sections[1])
		if relPart == `rel="last"` {
			// urlPart is enclosed in < >, so remove them.
			if len(urlPart) >= 2 && urlPart[0] == '<' && urlPart[len(urlPart)-1] == '>' {
				urlPart = urlPart[1 : len(urlPart)-1]
			}
			// Parse page parameter from the URL.
			idx := strings.Index(urlPart, "page=")
			if idx == -1 {
				continue
			}
			pageStr := urlPart[idx+5:]
			// pageStr might have other parameters; split by '&'
			pageStr = strings.Split(pageStr, "&")[0]
			if page, err := strconv.Atoi(pageStr); err == nil {
				return page
			}
		}
	}
	// If no "last" link is present, return 1.
	return 1
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

// GetCommits calls the commits endpoint and returns all the commits attached to the repositoryName.
// It supports optional "since" and "until" query parameters to filter commits.
func (c *Client) GetCommits(repositoryName, ownerName string, since, until *time.Time) ([]CommitResponse, error) {
	logr := c.logger.With(zap.String("method", "GetCommits"))
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

	logr.Debug("len of all commits after is: ", zap.Int("commits_count", len(allCommits)))
	return allCommits, nil
}

// GetCommitsNew fetches commits concurrently using a worker pool and sends each CommitResponse
// through commitCh. It also adds an authorization header if c.authToken is non-empty.
func (c *Client) GetCommitsNew(ctx context.Context, repositoryName, ownerName string, since, until *time.Time, pageSize int, commitCh chan<- CommitResponse) error {
	// Fetch page 1 synchronously.
	page := 1
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s/%s/commits", c.baseURL, ownerName, repositoryName), nil)
	if err != nil {
		return fmt.Errorf("failed to create request for page 1: %w", err)
	}
	// Add the Authorization header if token is provided.
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
	q.Set("per_page", fmt.Sprintf("%d", pageSize))
	q.Set("page", fmt.Sprintf("%d", page))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get page 1: %w", err)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to read page 1 response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API error on page 1: %s", string(bodyBytes))
	}
	var commitsPage []CommitResponse
	if err := sonic.Unmarshal(bodyBytes, &commitsPage); err != nil {
		return fmt.Errorf("failed to unmarshal page 1: %w", err)
	}
	for _, commit := range commitsPage {
		commitCh <- commit
	}
	// Parse Link header to get last page.
	lastPage := parseLastPage(resp.Header.Get("Link"))
	c.logger.Info("Fetched page 1", zap.Int("lastPage", lastPage))

	// If there's only one page, return.
	if lastPage <= 1 {
		c.logger.Info("No more pages of commits")
		close(commitCh) // Close the channel to signal completion.
		return nil
	}

	// Set up a worker pool to fetch pages 2..lastPage concurrently.
	numWorkers := 5
	jobs := make(chan int, lastPage-1)
	errCh := make(chan error, numWorkers)

	// Worker function.
	worker := func() {
		for pageNum := range jobs {
			select {
			case <-ctx.Done():
				errCh <- fmt.Errorf("context cancelled in worker")
				return
			default:
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s/%s/commits", c.baseURL, ownerName, repositoryName), nil)
			if err != nil {
				errCh <- fmt.Errorf("failed to create request for page %d: %w", pageNum, err)
				return
			}
			// Add the Authorization header if token is provided.
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
			q.Set("per_page", fmt.Sprintf("%d", pageSize))
			q.Set("page", fmt.Sprintf("%d", pageNum))
			req.URL.RawQuery = q.Encode()

			// Retry loop to handle rate limiting.
			attempt := 0
			maxRetries := 3
			var r *http.Response
			for {
				r, err = c.httpClient.Do(req)
				if err != nil {
					errCh <- fmt.Errorf("failed to get page %d: %w", pageNum, err)
					return
				}
				if remainingStr := r.Header.Get("X-RateLimit-Remaining"); remainingStr != "" {
					remaining, err := strconv.Atoi(remainingStr)
					if err == nil && remaining <= 0 {
						resetStr := r.Header.Get("X-RateLimit-Reset")
						if resetStr != "" {
							resetUnix, err := strconv.ParseInt(resetStr, 10, 64)
							if err == nil {
								sleepDuration := time.Until(time.Unix(resetUnix, 0))
								c.logger.Warn("Rate limit reached in worker", zap.Int("page", pageNum), zap.Duration("sleepDuration", sleepDuration))
								time.Sleep(sleepDuration)
								attempt++
								if attempt >= maxRetries {
									errCh <- fmt.Errorf("rate limit exceeded on page %d after max retries", pageNum)
									return
								}
								continue
							}
						}
					}
				}
				break
			}
			bodyBytes, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				errCh <- fmt.Errorf("failed to read page %d response: %w", pageNum, err)
				return
			}
			if r.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("GitHub API error on page %d: %s", pageNum, string(bodyBytes))
				return
			}
			var pageCommits []CommitResponse
			if err := sonic.Unmarshal(bodyBytes, &pageCommits); err != nil {
				errCh <- fmt.Errorf("failed to unmarshal page %d: %w", pageNum, err)
				return
			}
			// Send commits from this page.
			for _, commit := range pageCommits {
				commitCh <- commit
			}
		}
		errCh <- nil
	}

	// Launch workers.
	for i := 0; i < numWorkers; i++ {
		go worker()
	}

	// Enqueue jobs: pages 2 to lastPage.
	for p := 2; p <= lastPage; p++ {
		jobs <- p
	}
	close(jobs)

	// Wait for all workers.
	for i := 0; i < numWorkers; i++ {
		if wErr := <-errCh; wErr != nil {
			return wErr
		}
	}

	return nil
}

func (c *Client) GetRepositoryDetails(repositoryName, ownerName string) (*RepositoryResponse, error) {
	// logr := c.logger.With(zap.String("method", "GetRepositoryDetails"))
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
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var ghErr GitHubError
		if err := sonic.Unmarshal(body, &ghErr); err != nil {
			c.logger.Error("Unexpected status code", zap.Int("status code", resp.StatusCode), zap.Error(err))
		}

		// logr.Sugar().Infof("details of returned error: %v\n", ghErr)

		// logr.Debug("Github client failed", zap.Int("status_code", resp.StatusCode))

		return nil, errors.New(ghErr.Message)
	}

	var repo RepositoryResponse
	if err := sonic.Unmarshal(body, &repo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repository http response: %w", err)
	}

	return &repo, nil
}
