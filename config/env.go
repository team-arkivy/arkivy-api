package config

import (
	"log"

	"github.com/joho/godotenv"
)

// LoadEnv carga las variables del archivo .env.
// Si no se encuentra el archivo, avisa pero no detiene la ejecución,
// asumiendo que las variables vienen del entorno del sistema.
func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: No se encontró archivo .env, usando variables de entorno del sistema")
	}
}
