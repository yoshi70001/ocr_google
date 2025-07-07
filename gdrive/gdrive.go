// gdrive/gdrive.go
package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const tokenFile = "token.json"
const credentialsFile = "credentials.json"

// getClient utiliza un archivo de configuración para solicitar un token,
// luego lo guarda para usarlo en el futuro y devuelve el cliente HTTP.
func getClient(config *oauth2.Config) *http.Client {
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// getTokenFromWeb solicita un token desde la web y lo devuelve.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Ve a la siguiente URL en tu navegador y autoriza la aplicación: \n%v\n", authURL)
	fmt.Print("Ingresa el código de autorización: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("No se pudo leer el código de autorización: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("No se pudo obtener el token desde la web: %v", err)
	}
	return tok
}

// tokenFromFile recupera un token desde un archivo local.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken guarda un token en un archivo.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Guardando el token en el archivo: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("No se pudo guardar el token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// AuthenticateAndGetService crea y devuelve un servicio de Drive autenticado.
func AuthenticateAndGetService() (*drive.Service, error) {
	ctx := context.Background()
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el archivo de credenciales: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("no se pudo parsear el archivo de credenciales: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear el cliente de Drive: %v", err)
	}
	return srv, nil
}

// GetOrCreateFolder busca una carpeta en Drive o la crea si no existe. Devuelve su ID.
func GetOrCreateFolder(srv *drive.Service, folderName string) (string, error) {
	query := fmt.Sprintf("mimeType='application/vnd.google-apps.folder' and name='%s' and trashed=false", folderName)
	r, err := srv.Files.List().Q(query).PageSize(1).Fields("files(id)").Do()
	if err != nil {
		return "", fmt.Errorf("no se pudo buscar la carpeta: %v", err)
	}

	if len(r.Files) > 0 {
		log.Printf("Carpeta temporal '%s' encontrada con ID: %s", folderName, r.Files[0].Id)
		return r.Files[0].Id, nil
	}

	log.Printf("Creando carpeta temporal en Drive: '%s'", folderName)
	folderMetadata := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}
	folder, err := srv.Files.Create(folderMetadata).Fields("id").Do()
	if err != nil {
		return "", fmt.Errorf("no se pudo crear la carpeta: %v", err)
	}
	return folder.Id, nil
}

// ProcessImage realiza todo el proceso de OCR para una sola imagen.
func ProcessImage(srv *drive.Service, imagePath, textOutputPath, driveFolderID string) error {
	imageFileName := filepath.Base(imagePath)
	log.Printf("[+] Iniciando procesamiento para: %s", imageFileName)

	// Abrir el archivo de imagen local
	imgFile, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("no se pudo abrir la imagen local %s: %v", imagePath, err)
	}
	defer imgFile.Close()

	// 1. El "truco" de OCR: Crear un Google Doc a partir de la imagen
	// No es necesario subir la imagen primero, podemos hacerlo en un solo paso.
	log.Printf("    - Paso 1/3: Realizando OCR (creando Google Doc desde la imagen)...")
	docName := strings.TrimSuffix(imageFileName, filepath.Ext(imageFileName))
	docMetadata := &drive.File{
		Name:     docName,
		Parents:  []string{driveFolderID},
		MimeType: "application/vnd.google-apps.document", // La clave del OCR
	}

	doc, err := srv.Files.Create(docMetadata).Media(imgFile).Fields("id").Do()
	if err != nil {
		return fmt.Errorf("no se pudo crear el Google Doc para OCR: %v", err)
	}
	// Usamos defer para asegurarnos de que el doc se borre al final.
	defer func() {
		log.Printf("    - Limpiando Google Doc temporal (ID: %s)...", doc.Id)
		err := srv.Files.Delete(doc.Id).Do()
		if err != nil {
			log.Printf("ERROR: no se pudo borrar el doc temporal %s: %v", doc.Id, err)
		}
	}()

	// 2. Exportar y descargar el contenido del Doc como texto plano
	log.Printf("    - Paso 2/3: Descargando texto extraído...")
	res, err := srv.Files.Export(doc.Id, "text/plain").Download()
	if err != nil {
		return fmt.Errorf("no se pudo exportar el texto del Doc: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("no se pudo leer el cuerpo de la respuesta: %v", err)
	}

	// 3. Guardar el texto en un archivo local
	log.Printf("    - Paso 3/3: Guardando texto en %s", textOutputPath)
	err = os.WriteFile(textOutputPath, body, 0644)
	if err != nil {
		return fmt.Errorf("no se pudo guardar el archivo de texto: %v", err)
	}

	log.Printf("[✓] Procesamiento completado para: %s", imageFileName)
	return nil
}
