package zitadel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the HTTP client for the Zitadel v2 API
type Client struct {
	domain     string
	token      string
	httpClient *http.Client
	projectID  string
}

// NewClient creates a new Zitadel client
func NewClient(domain, serviceToken, projectID string) *Client {
	return &Client{
		domain:    domain,
		token:     serviceToken,
		projectID: projectID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// --- Request/Response types ---

// LoginRequest data sent by Angular
type LoginRequest struct {
	LoginName string `json:"loginName" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

// RegisterRequest sign-up form data
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SessionResponse returned to Angular
type SessionResponse struct {
	SessionID    string `json:"sessionId"`
	SessionToken string `json:"sessionToken"`
	UserID       string `json:"userId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	LoginName    string `json:"loginName,omitempty"`
}

// SessionInfo active session information
type SessionInfo struct {
	ID          string         `json:"id"`
	UserID      string         `json:"userId"`
	LoginName   string         `json:"loginName"`
	DisplayName string         `json:"displayName"`
	Factors     SessionFactors `json:"factors"`
}

type SessionFactors struct {
	User     *FactorUser     `json:"user,omitempty"`
	Password *FactorPassword `json:"password,omitempty"`
}

type FactorUser struct {
	VerifiedAt  string `json:"verifiedAt"`
	ID          string `json:"id"`
	LoginName   string `json:"loginName"`
	DisplayName string `json:"displayName"`
}

type FactorPassword struct {
	VerifiedAt string `json:"verifiedAt"`
}

// --- Client methods ---

// Login creates a session by verifying username + password in two steps
func (c *Client) Login(ctx context.Context, loginName, password string) (*SessionResponse, error) {
	// Step 1: Create session with user check
	createBody := map[string]any{
		"checks": map[string]any{
			"user": map[string]any{
				"loginName": loginName,
			},
		},
	}

	createResp, err := c.doRequest(ctx, "POST", "/v2/sessions", createBody)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}

	sessionID, ok := createResp["sessionId"].(string)
	if !ok {
		return nil, fmt.Errorf("sessionId not found in response")
	}

	// Step 2: Verify password
	updateBody := map[string]any{
		"checks": map[string]any{
			"password": map[string]any{
				"password": password,
			},
		},
	}

	updateResp, err := c.doRequest(ctx, "PATCH", "/v2/sessions/"+sessionID, updateBody)
	if err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	sessionToken, _ := updateResp["sessionToken"].(string)

	// Step 3: Get session info to return user data
	sessionInfo, err := c.GetSession(ctx, sessionID, sessionToken)
	if err != nil {
		// Session was created OK, return without extra data
		return &SessionResponse{
			SessionID:    sessionID,
			SessionToken: sessionToken,
		}, nil
	}

	return &SessionResponse{
		SessionID:    sessionID,
		SessionToken: sessionToken,
		UserID:       sessionInfo.UserID,
		DisplayName:  sessionInfo.DisplayName,
		LoginName:    sessionInfo.LoginName,
	}, nil
}

// Register creates a user and then logs in
func (c *Client) Register(ctx context.Context, username, password string) (*SessionResponse, string, error) {
	// Step 1: Create user in Zitadel
	userBody := map[string]any{
		"username": username,
		"profile": map[string]any{
			"givenName":  username, // Temporary, user can change it later
			"familyName": "-",
		},
		"email": map[string]any{
			"email": username,
			// Change when email verification is ready
			"isVerified": true,
		},
		"password": map[string]any{
			"password":       password,
			"changeRequired": false,
		},
	}

	createResp, err := c.doRequest(ctx, "POST", "/v2/users/human", userBody)
	if err != nil {
		return nil, "", fmt.Errorf("error creating user: %w", err)
	}

	userID, _ := createResp["userId"].(string)
	if userID == "" {
		return nil, "", fmt.Errorf("userId not found in response")
	}

	// Step 2: Assign default role
	if err := c.AssignRole(ctx, userID, RoleInvited); err != nil {
		// Not fatal — user was created, just doesn't have a role yet
		fmt.Printf("Warning: could not assign invited role to user %s: %v\n", userID, err)
	}

	// Step 3: Automatically create session
	session, err := c.Login(ctx, username, password)
	if err != nil {
		return nil, userID, fmt.Errorf("user created but error logging in: %w", err)
	}

	return session, userID, nil
}

// GetSession retrieves active session information
func (c *Client) GetSession(ctx context.Context, sessionID, sessionToken string) (*SessionInfo, error) {
	resp, err := c.doRequest(ctx, "GET", "/v2/sessions/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("error retrieving session: %w", err)
	}

	sessionData, ok := resp["session"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("session data not found")
	}

	info := &SessionInfo{
		ID: sessionID,
	}

	if factors, ok := sessionData["factors"].(map[string]any); ok {
		if user, ok := factors["user"].(map[string]any); ok {
			info.UserID, _ = user["id"].(string)
			info.LoginName, _ = user["loginName"].(string)
			info.DisplayName, _ = user["displayName"].(string)
		}

		if _, ok := factors["password"].(map[string]any); ok {
			info.Factors.Password = &FactorPassword{}
		}
	}

	return info, nil
}

// DeleteSession terminates a session (logout)
func (c *Client) DeleteSession(ctx context.Context, sessionID, sessionToken string) error {
	body := map[string]any{
		"sessionToken": sessionToken,
	}
	_, err := c.doRequest(ctx, "DELETE", "/v2/sessions/"+sessionID, body)
	if err != nil {
		return fmt.Errorf("error deleting session: %w", err)
	}
	return nil
}

// AssignRole assigns a project role to a user
func (c *Client) AssignRole(ctx context.Context, userID string, role Role) error {
	body := map[string]any{
		"projectId": c.projectID,
		"roleKeys":  []string{string(role)},
	}
	_, err := c.doRequest(ctx, "POST", "/management/v1/users/"+userID+"/grants", body)
	return err
}

// GetUserRoles retrieves a user's roles
func (c *Client) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	body := map[string]any{
		"queries": []map[string]any{
			{
				"userIdQuery": map[string]any{
					"userId": userID,
				},
			},
			{
				"projectIdQuery": map[string]any{
					"projectId": c.projectID,
				},
			},
		},
	}

	resp, err := c.doRequest(ctx, "POST", "/management/v1/users/grants/_search", body)
	if err != nil {
		return nil, err
	}

	roles := []string{}
	if result, ok := resp["result"].([]any); ok {
		for _, item := range result {
			if grant, ok := item.(map[string]any); ok {
				if roleKeys, ok := grant["roleKeys"].([]any); ok {
					for _, r := range roleKeys {
						if roleStr, ok := r.(string); ok {
							roles = append(roles, roleStr)
						}
					}
				}
			}
		}
	}

	return roles, nil
}

// StartIDPIntent starts the login flow with an external IDP (Google, GitHub)
func (c *Client) StartIDPIntent(ctx context.Context, idpID, successURL, failureURL string) (string, error) {
	body := map[string]any{
		"idpId": idpID,
		"urls": map[string]any{
			"successUrl": successURL,
			"failureUrl": failureURL,
		},
	}

	resp, err := c.doRequest(ctx, "POST", "/v2/idp_intents", body)
	if err != nil {
		return "", fmt.Errorf("error starting IDP intent: %w", err)
	}

	authURL, ok := resp["authorizationUrl"].(string)
	if !ok {
		return "", fmt.Errorf("authorizationUrl not found")
	}

	return authURL, nil
}

// CreateSessionWithIDP creates a session using a completed IDP intent
func (c *Client) CreateSessionWithIDP(ctx context.Context, userID, intentID, intentToken string) (*SessionResponse, error) {
	body := map[string]any{
		"checks": map[string]any{
			"user": map[string]any{
				"userId": userID,
			},
			"idpIntent": map[string]any{
				"idpIntentId":    intentID,
				"idpIntentToken": intentToken,
			},
		},
	}

	resp, err := c.doRequest(ctx, "POST", "/v2/sessions", body)
	if err != nil {
		return nil, fmt.Errorf("error creating session with IDP: %w", err)
	}

	sessionID, _ := resp["sessionId"].(string)
	sessionToken, _ := resp["sessionToken"].(string)

	return &SessionResponse{
		SessionID:    sessionID,
		SessionToken: sessionToken,
		UserID:       userID,
	}, nil
}

// --- Helper HTTP ---

func (c *Client) doRequest(ctx context.Context, method, path string, body any) (map[string]any, error) {
	url := c.domain + path

	var reqBody io.Reader
	if body != nil && method != "GET" {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Zitadel API error (%d): %s", resp.StatusCode, string(respBody))
	}

	// DELETE may return an empty body
	if len(respBody) == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return result, nil
}
