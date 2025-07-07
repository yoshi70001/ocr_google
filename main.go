// main.go
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yoshi70001/googleDocsOCR/gdrive"
)

const (
	imagesFolder    = "RGBImages"
	textsFolder     = "TXTImages"
	driveTempFolder = "Temp_OCR_Go"
)

func main() {
	log.Println("--- Iniciando Proceso de OCR con Google Drive y Go ---")

	// Crear carpetas locales si no existen
	if _, err := os.Stat(imagesFolder); os.IsNotExist(err) {
		os.Mkdir(imagesFolder, 0755)
		log.Fatalf("Carpeta '%s' creada. Por favor, pon tus imágenes ahí y vuelve a ejecutar.", imagesFolder)
	}
	if _, err := os.Stat(textsFolder); os.IsNotExist(err) {
		os.Mkdir(textsFolder, 0755)
	}

	// 1. Autenticar y obtener el servicio de Drive
	srv, err := gdrive.AuthenticateAndGetService()
	if err != nil {
		log.Fatalf("Fallo en la autenticación: %v", err)
	}
	log.Println("Autenticación exitosa.")

	// 2. Obtener o crear la carpeta temporal en Google Drive
	driveFolderID, err := gdrive.GetOrCreateFolder(srv, driveTempFolder)
	if err != nil {
		log.Fatalf("No se pudo obtener/crear la carpeta de Drive: %v", err)
	}

	// 3. Leer la lista de imágenes a procesar
	files, err := os.ReadDir(imagesFolder)
	if err != nil {
		log.Fatalf("No se pudo leer la carpeta de imágenes: %v", err)
	}

	var imagePaths []string
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".bmp" {
			imagePaths = append(imagePaths, filepath.Join(imagesFolder, file.Name()))
		}
	}

	if len(imagePaths) == 0 {
		log.Println("No se encontraron imágenes para procesar. Saliendo.")
		return
	}

	log.Printf("Se encontraron %d imágenes para procesar. Iniciando goroutines...", len(imagePaths))

	// 4. Procesar imágenes en paralelo usando goroutines
	var wg sync.WaitGroup
	for _, imgPath := range imagePaths {
		wg.Add(1) // Incrementar el contador del WaitGroup

		// Lanzar una goroutine por cada imagen
		go func(path string) {
			defer wg.Done() // Decrementar el contador cuando la goroutine termine

			baseName := filepath.Base(path)
			textName := strings.TrimSuffix(baseName, filepath.Ext(baseName)) + ".txt"
			textPath := filepath.Join(textsFolder, textName)

			err := gdrive.ProcessImage(srv, path, textPath, driveFolderID)
			if err != nil {
				// En un escenario real, podrías querer manejar este error de forma más robusta
				log.Printf("ERROR procesando %s: %v", baseName, err)
			}
		}(imgPath)
	}

	// 5. Esperar a que todas las goroutines terminen
	wg.Wait()

	log.Println("--- Todas las imágenes han sido procesadas. Proceso finalizado. ---")
}
