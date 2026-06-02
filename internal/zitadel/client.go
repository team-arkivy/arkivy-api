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

// Client es el cliente HTTP para la API v2 de Zitadel
type Client struct {
	domain     string
	token      string // PAT o service account token
	httpClient *http.Client
	projectID  string
}

// NewClient crea un nuevo cliente de Zitadel
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

// --- Tipos de request/response ---

// LoginRequest datos que envía Angular
type LoginRequest struct {
	LoginName string `json:"loginName" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

// RegisterRequest datos del formulario de sign-up
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SessionResponse lo que devolvemos a Angular
type SessionResponse struct {
	SessionID    string `json:"sessionId"`
	SessionToken string `json:"sessionToken"`
	UserID       string `json:"userId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	LoginName    string `json:"loginName,omitempty"`
}

// SessionInfo información de una sesión activa
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

// --- Métodos del cliente ---

// Login crea una sesión verificando username + password en dos pasos
func (c *Client) Login(ctx context.Context, loginName, password string) (*SessionResponse, error) {
	// Paso 1: Crear sesión con check de usuario
	createBody := map[string]any{
		"checks": map[string]any{
			"user": map[string]any{
				"loginName": loginName,
			},
		},
	}

	createResp, err := c.doRequest(ctx, "POST", "/v2/sessions", createBody)
	if err != nil {
		return nil, fmt.Errorf("error creando sesión: %w", err)
	}

	sessionID, ok := createResp["sessionId"].(string)
	if !ok {
		return nil, fmt.Errorf("sessionId no encontrado en respuesta")
	}

	// Paso 2: Verificar password
	updateBody := map[string]any{
		"checks": map[string]any{
			"password": map[string]any{
				"password": password,
			},
		},
	}

	updateResp, err := c.doRequest(ctx, "PATCH", "/v2/sessions/"+sessionID, updateBody)
	if err != nil {
		return nil, fmt.Errorf("contraseña inválida: %w", err)
	}

	sessionToken, _ := updateResp["sessionToken"].(string)

	// Paso 3: Obtener info de la sesión para devolver datos del usuario
	sessionInfo, err := c.GetSession(ctx, sessionID, sessionToken)
	if err != nil {
		// La sesión se creó OK, devolvemos sin los datos extra
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

// Register crea un usuario y luego inicia sesión
func (c *Client) Register(ctx context.Context, username, password string) (*SessionResponse, string, error) {
	// Paso 1: Crear usuario en Zitadel
	userBody := map[string]any{
		"username": username,
		"profile": map[string]any{
			"givenName":  username, // Temporal, el usuario puede cambiarlo después
			"familyName": "-",
		},
		"email": map[string]any{
			"email": username,
			// Cambiar para cuando la verficación por email este listo
			"isVerified": true,
		},
		"password": map[string]any{
			"password":       password,
			"changeRequired": false,
		},
	}

	createResp, err := c.doRequest(ctx, "POST", "/v2/users/human", userBody)
	if err != nil {
		return nil, "", fmt.Errorf("error creando usuario: %w", err)
	}

	userID, _ := createResp["userId"].(string)
	if userID == "" {
		return nil, "", fmt.Errorf("userId no encontrado en respuesta")
	}

	// Paso 2: Asignar rol por defecto
	if err := c.AssignRole(ctx, userID, RoleInvited); err != nil {
		// No es fatal — el usuario se creó, solo no tiene rol aún
		fmt.Printf("Aviso: no se pudo asignar rol invited al usuario %s: %v\n", userID, err)
	}

	// Paso 3: Crear sesión automáticamente
	session, err := c.Login(ctx, username, password)
	if err != nil {
		return nil, userID, fmt.Errorf("usuario creado pero error al iniciar sesión: %w", err)
	}

	return session, userID, nil
}

// GetSession obtiene la información de una sesión activa
func (c *Client) GetSession(ctx context.Context, sessionID, sessionToken string) (*SessionInfo, error) {
	resp, err := c.doRequest(ctx, "GET", "/v2/sessions/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo sesión: %w", err)
	}

	sessionData, ok := resp["session"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("datos de sesión no encontrados")
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

// DeleteSession termina una sesión (logout)
func (c *Client) DeleteSession(ctx context.Context, sessionID, sessionToken string) error {
	body := map[string]any{
		"sessionToken": sessionToken,
	}
	_, err := c.doRequest(ctx, "DELETE", "/v2/sessions/"+sessionID, body)
	if err != nil {
		return fmt.Errorf("error eliminando sesión: %w", err)
	}
	return nil
}

// AssignRole asigna un rol del proyecto a un usuario
func (c *Client) AssignRole(ctx context.Context, userID string, role Role) error {
	body := map[string]any{
		"projectId": c.projectID,
		"roleKeys":  []string{string(role)},
	}
	_, err := c.doRequest(ctx, "POST", "/management/v1/users/"+userID+"/grants", body)
	return err
}

// GetUserRoles obtiene los roles de un usuario
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

// StartIDPIntent inicia el flujo de login con un IDP externo (Google, GitHub)
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
		return "", fmt.Errorf("error iniciando IDP intent: %w", err)
	}

	authURL, ok := resp["authorizationUrl"].(string)
	if !ok {
		return "", fmt.Errorf("authorizationUrl no encontrada")
	}

	return authURL, nil
}

// CreateSessionWithIDP crea una sesión usando un IDP intent completado
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
		return nil, fmt.Errorf("error creando sesión con IDP: %w", err)
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

	// Para DELETE puede devolver body vacío
	if len(respBody) == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	return result, nil
}
