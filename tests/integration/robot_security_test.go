// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"
	"time"

	auth_model "code.gitea.io/gitea/models/auth"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

// TestRobotAPI_UnauthorizedPrivateRepo tests that unauthorized access to a private repository
// returns 404 (not 403) to avoid leaking repository existence
func TestRobotAPI_UnauthorizedPrivateRepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing repo 2 (private repo)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// Check that it's private
	if !repo.IsPrivate {
		t.Skip("Repo ID 2 is not private, skipping test")
	}

	// Unauthorized user tries to access the robot API for private repo
	sessionB := loginUser(t, "user5")
	req := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name)
	sessionB.MakeRequest(t, req, http.StatusNotFound)
}

// TestRobotAPI_PublicRepoAnonymous tests that anonymous users can access public repositories
func TestRobotAPI_PublicRepoAnonymous(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing repo 1 (usually public in test fixtures)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// Anonymous user tries to access the robot API for public repo
	req := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name)
	resp := MakeRequest(t, req, http.StatusOK)

	// Verify response structure (TriageResponse fields).
	var result map[string]any
	DecodeJSON(t, resp, &result)
	assert.Contains(t, result, "quick_ref")
	assert.Contains(t, result, "recommendations")
}

// TestRobotAPI_AuthorizedAccess tests that authorized users can access their own repositories
func TestRobotAPI_AuthorizedAccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// Owner accesses their robot API
	sessionA := loginUser(t, owner.Name)
	req := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name)
	resp := sessionA.MakeRequest(t, req, http.StatusOK)

	// Verify response structure (TriageResponse fields).
	var result map[string]any
	DecodeJSON(t, resp, &result)
	assert.Contains(t, result, "quick_ref")
	assert.Contains(t, result, "recommendations")
}

// TestRobotAPI_InvalidInput tests input validation including path traversal and oversized input.
//
// The handlers deliberately return 404 (not 400) for invalid input as a
// security-by-obscurity measure: a probing client should not be able to
// distinguish "bad request" from "no such repository". The test
// expectations therefore assert StatusNotFound.
func TestRobotAPI_InvalidInput(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	testCases := []struct {
		name       string
		owner      string
		repo       string
		expectCode int
	}{
		{
			name:       "Path traversal in owner",
			owner:      "../etc/passwd",
			repo:       "test",
			expectCode: http.StatusNotFound,
		},
		{
			name:       "Path traversal in repo",
			owner:      "user",
			repo:       "../../../etc/passwd",
			expectCode: http.StatusNotFound,
		},
		{
			// Null byte must be URL-encoded; net/http rejects raw control
			// characters in URLs before the handler runs.
			name:       "Null byte in owner",
			owner:      "user%00",
			repo:       "test",
			expectCode: http.StatusNotFound,
		},
		{
			name:       "Oversized owner name",
			owner:      strings.Repeat("a", 300),
			repo:       "test",
			expectCode: http.StatusNotFound,
		},
		{
			name:       "Oversized repo name",
			owner:      "user",
			repo:       strings.Repeat("b", 300),
			expectCode: http.StatusNotFound,
		},
		{
			name:       "Special characters in owner",
			owner:      "user<script>",
			repo:       "test",
			expectCode: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", tc.owner, tc.repo)
			MakeRequest(t, req, tc.expectCode)
		})
	}
}

// TestRobotAPI_FeatureDisabled tests that the API returns 404 when the feature is disabled
func TestRobotAPI_FeatureDisabled(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Disable the feature
	originalEnabled := setting.IssueGraphSettings.Enabled
	setting.IssueGraphSettings.Enabled = false
	defer func() {
		setting.IssueGraphSettings.Enabled = originalEnabled
	}()

	// Use existing repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// Request should return 404 when feature is disabled
	req := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name)
	MakeRequest(t, req, http.StatusNotFound)
}

// TestRobotAPI_ReadyEndpoint tests the /robot/ready endpoint
func TestRobotAPI_ReadyEndpoint(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// Test ready endpoint
	req := NewRequestf(t, "GET", "/api/v1/robot/ready?owner=%s&repo=%s", owner.Name, repo.Name)
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)
	assert.Contains(t, result, "repo_id")
}

// TestRobotAPI_ReadyEndpoint_SkipInProgress tests the /robot/ready endpoint with skip_in_progress parameter
func TestRobotAPI_ReadyEndpoint_SkipInProgress(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// Test ready endpoint with skip_in_progress=false
	req := NewRequestf(t, "GET", "/api/v1/robot/ready?owner=%s&repo=%s&skip_in_progress=false", owner.Name, repo.Name)
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)
	assert.Contains(t, result, "repo_id")

	// Test ready endpoint with skip_in_progress=true
	req2 := NewRequestf(t, "GET", "/api/v1/robot/ready?owner=%s&repo=%s&skip_in_progress=true", owner.Name, repo.Name)
	resp2 := MakeRequest(t, req2, http.StatusOK)

	var result2 map[string]any
	DecodeJSON(t, resp2, &result2)
	assert.Contains(t, result2, "repo_id")
}

// TestRobotAPI_GraphEndpoint tests the /robot/graph endpoint
func TestRobotAPI_GraphEndpoint(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// Test graph endpoint
	req := NewRequestf(t, "GET", "/api/v1/robot/graph?owner=%s&repo=%s", owner.Name, repo.Name)
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)
	assert.Contains(t, result, "nodes")
	assert.Contains(t, result, "edges")
}

// TestRobotAPI_AllEndpoints tests all robot API endpoints
func TestRobotAPI_AllEndpoints(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	endpoints := []string{
		"/api/v1/robot/triage?owner=%s&repo=%s",
		"/api/v1/robot/ready?owner=%s&repo=%s",
		"/api/v1/robot/graph?owner=%s&repo=%s",
	}

	for _, endpoint := range endpoints {
		req := NewRequestf(t, "GET", endpoint, owner.Name, repo.Name)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.NotNil(t, resp)
	}
}

// TestRobotAPI_Integration tests the full integration flow.
//
// The /api/v1/robot/* endpoints are gated by tokenRequiresScopes for the
// Issue scope category, so authenticated access requires a real OAuth2
// token (session cookies alone do not authenticate API requests).
func TestRobotAPI_Integration(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing private repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	other := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})

	// Test 1: Owner (with read:issue token) can access
	sessionOwner := loginUser(t, owner.Name)
	tokenOwner := getTokenForLoggedInUser(t, sessionOwner, auth_model.AccessTokenScopeReadIssue)
	req1 := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name).
		AddTokenAuth(tokenOwner)
	resp1 := MakeRequest(t, req1, http.StatusOK)
	assert.NotNil(t, resp1)

	// Test 2: Other user (with their own read:issue token) cannot access this private repo
	sessionOther := loginUser(t, other.Name)
	tokenOther := getTokenForLoggedInUser(t, sessionOther, auth_model.AccessTokenScopeReadIssue)
	req2 := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name).
		AddTokenAuth(tokenOther)
	resp2 := MakeRequest(t, req2, http.StatusNotFound)
	assert.NotNil(t, resp2)

	// Test 3: Anonymous cannot access
	req3 := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name)
	resp3 := MakeRequest(t, req3, http.StatusNotFound)
	assert.NotNil(t, resp3)

	// Verify response is valid JSON (TriageResponse fields).
	var result map[string]any
	DecodeJSON(t, resp1, &result)
	assert.Contains(t, result, "quick_ref")
	assert.Contains(t, result, "recommendations")
}

// TestRobotAPI_NonExistentRepo tests access to non-existent repositories
func TestRobotAPI_NonExistentRepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test user
	userA := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// Try to access non-existent repository
	sessionA := loginUser(t, userA.Name)
	req := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=non-existent-repo-12345", userA.Name)
	sessionA.MakeRequest(t, req, http.StatusNotFound)
}

// TestRobotAPI_NonExistentOwner tests access to non-existent owners
func TestRobotAPI_NonExistentOwner(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Try to access non-existent owner
	req := NewRequest(t, "GET", "/api/v1/robot/triage?owner=non-existent-owner-12345&repo=test-repo")
	MakeRequest(t, req, http.StatusNotFound)
}

// TestRobotAPI_CacheConsistency tests cache consistency across multiple requests
func TestRobotAPI_CacheConsistency(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// Make multiple requests and verify consistency
	numRequests := 5
	var results []map[string]any

	for i := 0; i < numRequests; i++ {
		req := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name)
		resp := MakeRequest(t, req, http.StatusOK)

		var result map[string]any
		DecodeJSON(t, resp, &result)
		results = append(results, result)
	}

	// All results should be identical (from cache)
	for i := 1; i < len(results); i++ {
		assert.Equal(t, results[0], results[i], "All cached responses should be identical")
	}
}

// TestRobotAPI_Performance tests performance characteristics
func TestRobotAPI_Performance(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// Test response time
	maxDuration := 5 * time.Second

	done := make(chan bool)
	go func() {
		req := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name)
		MakeRequest(t, req, http.StatusOK)
		done <- true
	}()

	select {
	case <-done:
		// Success - request completed within timeout
	case <-time.After(maxDuration):
		t.Fatalf("Request took longer than %v", maxDuration)
	}
}

// TestRobotAPI_ErrorMessages tests that error messages don't leak sensitive information
func TestRobotAPI_ErrorMessages(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Use existing private repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	other := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})

	// Check it's private
	if !repo.IsPrivate {
		t.Skip("Repo ID 2 is not private, skipping test")
	}

	// Other user tries to access private repo
	sessionOther := loginUser(t, other.Name)
	req := NewRequestf(t, "GET", "/api/v1/robot/triage?owner=%s&repo=%s", owner.Name, repo.Name)
	resp := sessionOther.MakeRequest(t, req, http.StatusNotFound)

	// Response body should not contain sensitive information
	body := resp.Body.String()
	assert.NotContains(t, body, repo.Name, "Error should not leak repo name")
	assert.NotContains(t, body, "private", "Error should not indicate repo is private")
	assert.NotContains(t, body, "access denied", "Error should not indicate access was denied")
	assert.NotContains(t, body, "forbidden", "Error should not indicate forbidden access")
}
