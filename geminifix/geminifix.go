// geminifix/geminifix.go
package geminifix

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func NewClient(ctx context.Context) (*genai.Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("la variable de entorno GEMINI_API_KEY no está configurada")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("error al crear el cliente de Gemini: %w", err)
	}
	return client, nil
}

// buildBatchAnimePrompt construye un prompt para corregir un lote de subtítulos.
func buildBatchAnimePrompt(texts []string) string {
	var numberedTexts strings.Builder
	for i, text := range texts {
		// Formateamos cada línea con un identificador único.
		numberedTexts.WriteString(fmt.Sprintf("LÍNEA %d: %s\n", i, text))
	}

	return fmt.Sprintf(`
Eres un experto transcriptor y corrector de subtítulos para anime. Tu tarea es corregir errores ortográficos y de dicción del siguiente lote de textos, que provienen de subtítulos consecutivos. Debes mejorar la redacción para que suene natural en español, pero SIN cambiar el significado original.

REGLAS IMPORTANTES:
1.  **Respeta los nombres propios** y **sufijos honoríficos** japoneses (-san, -chan, etc.).
2.  Mantén el tono y la consistencia del diálogo a lo largo de todas las líneas.
3.  **Devuelve el resultado como un listado**, manteniendo el formato "LÍNEA <NÚMERO>: <TEXTO CORREGIDO>" para cada línea que te envié.
4.  **DEBES DEVOLVER EXACTAMENTE EL MISMO NÚMERO DE LÍNEAS QUE RECIBISTE.** Si una línea no necesita cambios, repítela tal cual.
5.  **NO AÑADAS NINGÚN OTRO TEXTO.** Solo la lista numerada de líneas corregidas.

Lote de textos a corregir:
%s
`, numberedTexts.String())
}

// CorrectTextBatch utiliza Gemini para corregir un lote de textos.
func CorrectTextBatch(ctx context.Context, client *genai.Client, batchToCorrect []string) ([]string, error) {
	if len(batchToCorrect) == 0 {
		return []string{}, nil
	}

	model := client.GenerativeModel("gemini-1.5-flash-latest")
	prompt := buildBatchAnimePrompt(batchToCorrect)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("error al generar contenido con Gemini: %w", err)
	}

	// ... (código para extraer el texto de la respuesta de Gemini) ...
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini no devolvió candidatos")
	}
	rawResponse, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, fmt.Errorf("la respuesta de Gemini no es de tipo texto")
	}

	// --- PARSEAR LA RESPUESTA DE GEMINI ---
	// Ahora debemos procesar la respuesta para extraer cada línea corregida.
	correctedBatch := make([]string, len(batchToCorrect))
	lines := strings.Split(string(rawResponse), "\n")

	parsedCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Buscamos el formato "LÍNEA <NÚMERO>: <TEXTO>"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue // Línea mal formateada, la ignoramos
		}

		var index int
		// Extraemos el número de "LÍNEA N"
		_, err := fmt.Sscanf(parts[0], "LÍNEA %d", &index)
		if err != nil {
			continue // No pudimos parsear el número, ignoramos la línea
		}

		// Verificamos que el índice esté dentro de los límites de nuestro slice
		if index >= 0 && index < len(correctedBatch) {
			correctedBatch[index] = strings.TrimSpace(parts[1])
			parsedCount++
		}
	}

	// Verificación de seguridad: si Gemini no devolvió todas las líneas, es un problema.
	if parsedCount != len(batchToCorrect) {
		log.Printf("[!] ADVERTENCIA: Gemini devolvió %d líneas, pero se esperaban %d. Puede haber subtítulos vacíos.", parsedCount, len(batchToCorrect))
		// Rellenamos las líneas que falten con el texto original para no perder subtítulos.
		for i, text := range correctedBatch {
			if text == "" {
				correctedBatch[i] = batchToCorrect[i]
				log.Printf("    - Rellenando línea faltante %d con el texto original.", i)
			}
		}
	}

	return correctedBatch, nil
}
