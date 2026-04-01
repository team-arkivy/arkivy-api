package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Carga el archivo .env ubicado en el mismo directorio
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error cargando el archivo .env:", err)
	}

	// Lee las variables de entorno
	port := os.Getenv("PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")

	fmt.Println("--- Ejemplo godotenv cargado exitosamente ---")
	fmt.Printf("Servidor corriendo en el puerto: %s\n", port)
	fmt.Printf("Usuario de BD: %s\n", dbUser)
	fmt.Printf("Contraseña (oculta por seguridad): %s***\n", string(dbPass[0]))
}
