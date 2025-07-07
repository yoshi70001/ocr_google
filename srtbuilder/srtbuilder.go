// srtbuilder/srtbuilder.go
package srtbuilder

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SubtitleBlock representa una única entrada en un archivo SRT.
type SubtitleBlock struct {
	Sequence  int
	StartTime string
	EndTime   string
	Text      string
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
	parts := strings.Split(base, "___")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("el nombre de archivo no contiene el separador '__': %s", filename)
	}

	startTime, err := formatSrtTime(parts[0])
	if err != nil {
		return "", "", err
	}

	endTime, err := formatSrtTime(parts[1])
	if err != nil {
		return "", "", err
	}

	return startTime, endTime, nil
}

// CreateSrtFromTextFiles lee una carpeta de archivos .txt, los ordena,
// y construye un archivo .srt.
func CreateSrtFromTextFiles(textFolder, outputSrtFile string) error {
	log.Println("--- Iniciando construcción de archivo SRT ---")

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
