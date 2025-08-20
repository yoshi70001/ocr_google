// main.go
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/yoshi70001/googleDocsOCR/gdrive"
	"github.com/yoshi70001/googleDocsOCR/geminifix"
	"github.com/yoshi70001/googleDocsOCR/srtbuilder"
)

const (
	imagesFolder    = "RGBImages"
	textsFolder     = "TXTImages"
	driveTempFolder = "Temp_OCR_Go"
	outputSrtFile   = "subtitulo.srt"
)

var version = "development"

func main() {
	log.Printf("googleDocsOCR version %s", version)

       useGemini := flag.Bool("use-gemini", false, "Activar corrección de texto con Gemini")
       flag.Parse()

       ctx := context.Background()

       // Inicializar cliente de Gemini solo si se solicita
       var geminiClient *genai.Client
       if *useGemini {
	       if os.Getenv("GEMINI_API_KEY") != "" {
		       var err error
		       geminiClient, err = geminifix.NewClient(ctx)
		       if err != nil {
			       log.Fatalf("Fallo al inicializar el cliente de Gemini: %v", err)
		       }
		       defer geminiClient.Close()
		       log.Println("✓ Cliente de Gemini inicializado.")
	       } else {
		       log.Println("[!] ADVERTENCIA: No se encontró la GEMINI_API_KEY. Se procederá sin corrección de IA.")
	       }
       } else {
	       log.Println("[!] Gemini desactivado. No se realizará corrección de IA.")
       }

       // --- PASO 1: PROCESAMIENTO OCR ---
       log.Println("===== INICIANDO PASO 1: EXTRACCIÓN DE TEXTO (OCR) =====")

	// Crear carpetas locales si no existen
	if _, err := os.Stat(imagesFolder); os.IsNotExist(err) {
		os.Mkdir(imagesFolder, 0755)
		log.Fatalf("Carpeta '%s' creada. Por favor, pon tus imágenes ahí y vuelve a ejecutar.", imagesFolder)
	}
	if _, err := os.Stat(textsFolder); os.IsNotExist(err) {
		os.Mkdir(textsFolder, 0755)
	}

	srv, err := gdrive.AuthenticateAndGetService()
	if err != nil {
		log.Fatalf("Fallo en la autenticación: %v", err)
	}
	log.Println("✓ Autenticación exitosa.")

	driveFolderID, err := gdrive.GetOrCreateFolder(srv, driveTempFolder)
	if err != nil {
		log.Fatalf("No se pudo obtener/crear la carpeta de Drive: %v", err)
	}

	// Leer y ordenar las imágenes a procesar
	files, err := os.ReadDir(imagesFolder)
	if err != nil {
		log.Fatalf("No se pudo leer la carpeta de imágenes: %v", err)
	}

	var imagePaths []string
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			imagePaths = append(imagePaths, file.Name())
		}
	}
	sort.Strings(imagePaths)

	if len(imagePaths) == 0 {
		log.Println("No se encontraron imágenes para procesar.")
	} else {
		log.Printf("Se procesarán %d imágenes. Iniciando goroutines...", len(imagePaths))
		startTime := time.Now()

		// --- CONTROL DE CONCURRENCIA ---
		const maxConcurrency = 5 // Limita a 5 subidas simultáneas
		semaphore := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup
		// -------------------------------

		for _, imgFilename := range imagePaths {
			wg.Add(1)
			semaphore <- struct{}{} // Adquiere un "slot"

			go func(filename string) {
				defer wg.Done()
				defer func() { <-semaphore }() // Libera el "slot" al final

				fullImagePath := filepath.Join(imagesFolder, filename)
				textFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".txt"
				fullTextPath := filepath.Join(textsFolder, textFilename)

				if _, err := os.Stat(fullTextPath); err == nil {
					log.Printf("[SKIP] El archivo de texto para '%s' ya existe. Saltando OCR.", filename)
					return
				}

				err := gdrive.ProcessImage(srv, fullImagePath, fullTextPath, driveFolderID)
				if err != nil {
					log.Printf("ERROR procesando %s: %v", filename, err)
				}
			}(imgFilename)
		}
		wg.Wait()
		log.Printf("✓ OCR completado. Tiempo total: %s", time.Since(startTime))
	}

	log.Println("===== PASO 1 COMPLETADO =====")
	log.Println("") // Línea en blanco para separar

	// --- PASO 2: CONSTRUCCIÓN DEL SRT ---
	log.Println("===== INICIANDO PASO 2: CREACIÓN DE ARCHIVO SRT =====")

	err = srtbuilder.CreateSrtFromTextFiles(textsFolder, outputSrtFile, geminiClient)
	if err != nil {
		log.Fatalf("Fallo al crear el archivo SRT: %v", err)
	}

	log.Println("===== PROCESO FINALIZADO CON ÉXITO =====")
}
