// srtbuilder/srtbuilder.go
package srtbuilder

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/yoshi70001/googleDocsOCR/geminifix"
)

// SubtitleBlock representa una única entrada en un archivo SRT.
type SubtitleBlock struct {
	Sequence  int
	StartTime string
	EndTime   string
	Text      string
}

// processBatch es una nueva función de ayuda para manejar la llamada a la IA.
func processBatch(ctx context.Context, geminiClient *genai.Client, textBatch []string) []string {
	log.Printf("  [AI] Enviando lote de %d textos a Gemini para corrección...", len(textBatch))

	// Reintentos simples
	var correctedBatch []string
	var geminiErr error
	for attempt := range 3 {
		correctedBatch, geminiErr = geminifix.CorrectTextBatch(ctx, geminiClient, textBatch)
		if geminiErr == nil {
			log.Printf("  [✓] Lote procesado por Gemini.")
			return correctedBatch // Éxito
		}
		log.Printf("  [!] ADVERTENCIA: Intento %d de Gemini falló para el lote: %v. Reintentando...", attempt+1, geminiErr)
		time.Sleep(2 * time.Second)
	}

	log.Printf("  [!] ERROR: Todos los intentos de Gemini fallaron para el lote. Usando textos originales.")
	return textBatch // Devolvemos el lote original si todo falla
}

func cleanOcrText(rawText string) string {
	// Dividimos el texto en un slice de líneas usando el salto de línea como separador.
	lines := strings.Split(rawText, "\n")

	// Medida de seguridad: Si hay 2 o menos líneas, significa que el texto es muy corto
	// o está vacío. Devolverlo como está o devolver una cadena vacía es más seguro
	// que intentar acceder a un índice que no existe (lo que causaría un pánico).
	if len(lines) <= 2 {
		// Podemos decidir qué hacer. Devolver la última línea (si existe) podría ser una opción.
		// O simplemente devolver una cadena vacía si es probable que sea todo ruido.
		// Vamos a unir todo lo que haya, que en este caso será poco o nada.
		return strings.TrimSpace(rawText)
	}

	// Seleccionamos el sub-slice que va desde el tercer elemento (índice 2) hasta el final.
	relevantLines := lines[2:]

	// Unimos las líneas relevantes de nuevo en una sola cadena de texto,
	// usando el salto de línea para preservar los párrafos.
	cleanedText := strings.Join(relevantLines, "\n")

	// Finalmente, eliminamos cualquier espacio en blanco al principio o al final
	// que pudiera haber quedado.
	return strings.TrimSpace(cleanedText)
}

// formatSrtTime convierte "HH_MM_SS_FFF" a "HH:MM:SS,FFF".
func formatSrtTime(t string) (string, error) {
	parts := strings.Split(t, "_")
	if len(parts) != 4 {
		return "", fmt.Errorf("formato de tiempo inválido: %s", t)
	}
	return fmt.Sprintf("%s:%s:%s,%s", parts[0], parts[1], parts[2], parts[3]), nil
}

// parseFilename extrae los tiempos de inicio y fin del nombre de archivo.
func parseFilename(filename string) (string, string, error) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(base, "__")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("el nombre de archivo no contiene el separador '__': %s", filename)
	}

	startTime, err := formatSrtTime(parts[0])
	if err != nil {
		return "", "", err
	}
	auxend := strings.Split(parts[1], "_")
	if len(auxend) > 4 {
		auxend = auxend[:len(auxend)-1]
		parts[1] = strings.Join(auxend, "_")
	}
	endTime, err := formatSrtTime(parts[1])
	if err != nil {
		return "", "", err
	}

	return startTime, endTime, nil
}

// CreateSrtFromTextFiles lee una carpeta de archivos .txt, los ordena,
// y construye un archivo .srt.
func CreateSrtFromTextFiles(textFolder, outputSrtFile string, geminiClient *genai.Client) error {
	log.Println("--- Iniciando construcción de archivo SRT ---")
	batchSize := 100
	ctx := context.Background()

	// 1. Leer y ordenar los archivos de texto. La ordenación es crucial.
	files, err := os.ReadDir(textFolder)
	if err != nil {
		return fmt.Errorf("no se pudo leer la carpeta de textos '%s': %w", textFolder, err)
	}

	var textFilenames []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
			textFilenames = append(textFilenames, file.Name())
		}
	}
	sort.Strings(textFilenames)

	if len(textFilenames) == 0 {
		return fmt.Errorf("no se encontraron archivos .txt en la carpeta '%s'", textFolder)
	}

	log.Printf("✓ Se encontraron y ordenaron %d archivos de texto.", len(textFilenames))

	// 2. Construir la lista de bloques de subtítulos
	var blocks []SubtitleBlock
	for i, filename := range textFilenames {
		// Parsear el nombre del archivo para los tiempos
		start, end, err := parseFilename(filename)
		if err != nil {
			log.Printf("  [!] ADVERTENCIA: Saltando archivo con nombre inválido '%s': %v", filename, err)
			continue
		}

		// Leer el contenido del archivo de texto
		content, err := os.ReadFile(filepath.Join(textFolder, filename))
		if err != nil {
			log.Printf("  [!] ADVERTENCIA: No se pudo leer el archivo '%s': %v. Usando texto vacío.", filename, err)
			content = []byte("[ERROR DE LECTURA]")
		}

		blocks = append(blocks, SubtitleBlock{
			Sequence:  i + 1,
			StartTime: start,
			EndTime:   end,
			Text:      cleanOcrText(string(content)),
		})
	}
	// Ahora, si tenemos cliente de IA, procesamos los textos en lotes
	if geminiClient != nil {
		for i := 0; i < len(blocks); i += batchSize {
			end := min(i+batchSize, len(blocks))

			// Extraemos el lote de textos originales
			currentBatchBlocks := blocks[i:end]
			originalTextBatch := make([]string, len(currentBatchBlocks))
			for j, block := range currentBatchBlocks {
				originalTextBatch[j] = block.Text
			}

			// Procesamos el lote con Gemini
			correctedTextBatch := processBatch(ctx, geminiClient, originalTextBatch)

			// Actualizamos los bloques con los textos corregidos
			if len(correctedTextBatch) == len(currentBatchBlocks) {
				for j := range currentBatchBlocks {
					blocks[i+j].Text = correctedTextBatch[j]
				}
			} else {
				log.Printf("[!] ERROR CRÍTICO: El tamaño del lote devuelto (%d) no coincide con el enviado (%d). Se usarán textos originales para este lote.", len(correctedTextBatch), len(currentBatchBlocks))
			}
		}
	}

	// 3. Escribir el archivo .srt final
	log.Printf("✍️  Escribiendo archivo final: %s", outputSrtFile)
	file, err := os.Create(outputSrtFile)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo SRT: %w", err)
	}
	defer file.Close()

	for _, block := range blocks {
		// Si el texto está vacío, opcionalmente podemos saltarlo o poner un placeholder.
		// Aquí lo incluiremos para mantener la secuencia.
		if block.Text == "" {
			block.Text = "..."
		}

		srtEntry := fmt.Sprintf("%d\n%s --> %s\n%s\n\n",
			block.Sequence,
			block.StartTime,
			block.EndTime,
			block.Text)

		if _, err := file.WriteString(srtEntry); err != nil {
			// Devolvemos el primer error que encontremos al escribir
			return fmt.Errorf("error al escribir en el archivo SRT: %w", err)
		}
	}

	log.Println("🎉 ¡Archivo SRT creado exitosamente!")
	return nil
}
