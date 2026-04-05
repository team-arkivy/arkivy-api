package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"arkivy-api/Internal/arkivy/test1"
)

var (
	mongoClient *mongo.Client
	oauthConfig *oauth2.Config
	userHandler *test1.UserHandler
)

func main() {
	// 1. Cargar configuración
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: No se encontró archivo .env, usando variables de entorno del sistema")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	// 2. Configurar conexión a MongoDB
	connectToMongo(mongoURI)
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	// Inicializar la base de datos "arkivy" y el módulo de Clean Architecture
	db := mongoClient.Database("arkivy")
	userModule := test1.InitUserModule(db)
	userHandler = userModule.Handler

	// 3. Configurar OAuth2 de Google
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// 4. Inicializar servidor Gin
	r := gin.Default()

	// Configurar CORS para permitir requests desde Angular
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Grupo API global
	api := r.Group("/api")
	{
		// Ruta base para test de API
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"mensaje": "Servidor activo con MongoDB, CORS habilitado y rutas en /api"})
		})

		// Auth genérica (mock temporal para el frontend Angular)
		api.POST("/auth/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"accessToken": "mock-token-for-now",
				"user":        gin.H{"id": "1", "email": "test@test.com", "name": "Usuario Test Angular", "role": "USER"},
			})
		})
		api.POST("/auth/register", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Registro mock exitoso",
			})
		})

		// Rutas de autenticación con Google
		authGoogle := api.Group("/auth/google")
		{
			authGoogle.GET("/login", handleGoogleLogin)
			authGoogle.GET("/callback", handleGoogleCallback)
		}
	}

	// Iniciar servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Iniciando servidor en el puerto: %s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Error iniciando el servidor:", err)
	}
}

func connectToMongo(uri string) {
	var err error

	// Crear cliente y conectarse al servidor
	opts := options.Client().ApplyURI(uri)
	mongoClient, err = mongo.Connect(opts)
	if err != nil {
		log.Fatal("Error al crear cliente MongoDB:", err)
	}

	// Hacer ping para confirmar que la conexión funciona
	err = mongoClient.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal("Error al hacer ping a MongoDB:", err)
	}

	fmt.Println("Conectado exitosamente a MongoDB Compass!")
}

// Genera la URL de Google y redirige al usuario
func handleGoogleLogin(c *gin.Context) {
	// state debería ser un valor aleatorio generado por sesión para evitar ataques CSRF.
	// Para este ejemplo usamos un string fijo.
	state := "estado-aleatorio-para-seguridad"
	url := oauthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// Recibe la redirección de Google con el "code" y obtiene los datos del usuario
func handleGoogleCallback(c *gin.Context) {
	state := c.Query("state")
	if state != "estado-aleatorio-para-seguridad" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Estado (state) inválido"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se recibió el código de autorización"})
		return
	}

	// Intercambiar el código por un token de acceso
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al intercambiar el token: " + err.Error()})
		return
	}

	// Usar el token para obtener info del usuario desde la API de Google
	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener datos del usuario: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	// Decodificar la respuesta JSON de Google a un mapa
	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar JSON de usuario"})
		return
	}

	// Llamar al handler estructurado (Clean Architecture) para que guarde o inicie sesión
	user, err := userHandler.ProcessGoogleUser(context.Background(), userInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error registrando usuario en base de datos: " + err.Error()})
		return
	}

	redirectURL, err := buildFrontendGoogleRedirectURL(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error construyendo redirección al frontend: " + err.Error()})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func buildFrontendGoogleRedirectURL(user *test1.Usuario) (string, error) {
	frontendBaseURL := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	if frontendBaseURL == "" {
		frontendBaseURL = "http://localhost:4200"
	}

	userPayload, err := json.Marshal(user)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("googleSuccess", "1")
	params.Set("token", "google-"+user.GoogleID)
	params.Set("user", string(userPayload))

	return fmt.Sprintf("%s/login?%s", frontendBaseURL, params.Encode()), nil
}
