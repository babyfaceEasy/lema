package githubapi_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/internal/integrations/githubapi"
	mock_githubapi "github.com/babyfaceeasy/lema/internal/integrations/githubapi/mock_httpclient"
	"github.com/bytedance/sonic"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetRepositoryDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock HTTP client and a no-op logger.
	mockHttpClient := mock_githubapi.NewMockHttpClient(ctrl)
	logger := zap.NewNop()

	baseURL := "https://api.github.com/repos"
	ownerName := "chromium"
	repoName := "chromium"

	mockConfig := config.Config{}

	// Create our client with the mocked HTTP client.
	client := githubapi.NewClient(baseURL, mockHttpClient, logger, &mockConfig)

	// Build an expected RepositoryResponse.
	expectedResponse := githubapi.RepositoryResponse{
		Name:                "chromium",
		URL:                 "https://api.github.com/repos/chromium/chromium",
		Description:         "The official GitHub mirror of the Chromium source",
		ProgrammingLanguage: "C++",
		ForksCount:          7408,
		OpenIssuesCount:     118,
		WatchersCount:       20112,
		StarsCount:          20112,
	}

	// Marshal expected response to JSON.
	jsonBytes, err := sonic.Marshal(expectedResponse)
	require.NoError(t, err)

	expectedURL := fmt.Sprintf("%s/%s/%s", baseURL, ownerName, repoName)
	mockHttpClient.
		EXPECT().
		Do(gomock.AssignableToTypeOf(&http.Request{})).
		DoAndReturn(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, expectedURL, req.URL.String())
			require.Equal(t, http.MethodGet, req.Method)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(jsonBytes)),
			}, nil
		})

	// Call the function under test.
	repoResponse, err := client.GetRepositoryDetails(repoName, ownerName)
	require.NoError(t, err)
	require.Equal(t, expectedResponse, *repoResponse)
}

func TestGetCommits(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHttpClient := mock_githubapi.NewMockHttpClient(ctrl)
	logger := zap.NewNop()

	baseURL := "https://api.github.com/repos"
	ownerName := "chromium"
	repoName := "chromium"

	mockConfig := config.Config{}

	client := githubapi.NewClient(baseURL, mockHttpClient, logger, &mockConfig)

	// Define sample since and until times.
	sinceTime := time.Now().Add(-24 * time.Hour)
	untilTime := time.Now()

	// Build an expected commits response.
	expectedCommits := []githubapi.CommitResponse{
		{
			SHA: "abc123",
			URL: "https://api.github.com/repos/chromium/chromium/commits/abc123",
			Commit: githubapi.CommitDetail{
				Author: githubapi.Person{
					Name:  "John Doe",
					Email: "john@example.com",
					Date:  time.Date(2025, time.March, 13, 23, 9, 53, 908690000, time.Local),
				},
				Committer: githubapi.Person{
					Name:  "John Doe",
					Email: "john@example.com",
					Date:  time.Date(2025, time.March, 13, 23, 9, 53, 908690000, time.Local),
				},
				Message:      "Initial commit",
				URL:          "https://api.github.com/repos/chromium/chromium/commits/abc123",
				CommentCount: 0,
			},
		},
	}

	jsonBytes, err := sonic.Marshal(expectedCommits)
	require.NoError(t, err)

	expectedURL := fmt.Sprintf("%s/%s/%s/commits", baseURL, ownerName, repoName)
	mockHttpClient.
		EXPECT().
		Do(gomock.AssignableToTypeOf(&http.Request{})).
		DoAndReturn(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, expectedURL, req.URL.Scheme+"://"+req.URL.Host+req.URL.Path)
			require.Equal(t, http.MethodGet, req.Method)
			// Check query parameters.
			q := req.URL.Query()
			require.Equal(t, sinceTime.UTC().Format(time.RFC3339), q.Get("since"))
			require.Equal(t, untilTime.UTC().Format(time.RFC3339), q.Get("until"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(jsonBytes)),
			}, nil
		})

	actualCommits, err := client.GetCommits(repoName, ownerName, &sinceTime, &untilTime)
	require.NoError(t, err)
	require.Len(t, actualCommits, len(expectedCommits))

	// Compare each commit field by field.
	for i, expected := range expectedCommits {
		actual := actualCommits[i]
		require.Equal(t, expected.SHA, actual.SHA)
		require.Equal(t, expected.URL, actual.URL)
		// Compare commit details.
		require.Equal(t, expected.Commit.Message, actual.Commit.Message)
		require.Equal(t, expected.Commit.URL, actual.Commit.URL)
		require.Equal(t, expected.Commit.CommentCount, actual.Commit.CommentCount)
		// Compare Author and Committer fields.
		require.Equal(t, expected.Commit.Author.Name, actual.Commit.Author.Name)
		require.Equal(t, expected.Commit.Author.Email, actual.Commit.Author.Email)
		require.WithinDuration(t, expected.Commit.Author.Date, actual.Commit.Author.Date, time.Second)
		require.Equal(t, expected.Commit.Committer.Name, actual.Commit.Committer.Name)
		require.Equal(t, expected.Commit.Committer.Email, actual.Commit.Committer.Email)
		require.WithinDuration(t, expected.Commit.Committer.Date, actual.Commit.Committer.Date, time.Second)
	}
}
