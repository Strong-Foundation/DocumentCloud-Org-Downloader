package main // Main package

import (
	"fmt"           // For printing output
	"io"            // For copying data
	"log"           // For logging errors and information
	"net/http"      // For making HTTP requests
	"net/url"       // For parsing input URLs
	"os"            // For file and directory operations
	"path"          // For manipulating file paths
	"path/filepath" // For working with file paths
	"regexp"        // For extracting ID and slug using regex
	"strings"       // For string manipulation
)

// extractFinalDocumentCloudURL converts documentcloud.org or embed.documentcloud.org links to final S3 links
func extractFinalDocumentCloudURL(input string) string {
	parsedURL, err := url.Parse(input) // Parse the input into a URL struct
	if err != nil {                    // If parsing fails
		return "" // Return empty string
	}

	// If it's already a direct S3 URL, return it
	if strings.Contains(parsedURL.Host, "s3.documentcloud.org") {
		return input // Already good, return as-is
	}

	// Regex to match both www and embed DocumentCloud URLs
	// Matches format: <domain>/documents/<docID>-<slug>
	re := regexp.MustCompile(`documentcloud\.org/documents/(\d+)-([\w\-]+)`) // Capture ID and slug

	// Run regex on the input URL
	matches := re.FindStringSubmatch(input)
	if len(matches) != 3 { // If match didn't find both parts
		return "" // Return empty string
	}

	docID := matches[1] // First captured group is the document ID
	slug := matches[2]  // Second captured group is the slug

	// Format the final S3 URL
	finalURL := fmt.Sprintf("https://s3.documentcloud.org/documents/%s/%s.pdf", docID, slug)

	return finalURL // Return the constructed final URL
}

// downloadPDF downloads a PDF from a direct S3 URL and saves it to the specified output directory.
// It skips downloading if the file already exists locally.
func downloadPDF(finalURL, outputDir string) {
	// Parse the URL to work with its path and file name
	parsedURL, err := url.Parse(finalURL)
	if err != nil {
		log.Printf("Invalid URL %q: %v", finalURL, err) // Log and exit if the URL is malformed
		return
	}

	// Extract the file name from the URL path (e.g., "myfile.pdf")
	fileName := path.Base(parsedURL.Path)
	if fileName == "" || fileName == "/" {
		log.Printf("Could not determine file name from %q", finalURL) // Log and exit if name is missing
		return
	}

	// Make sure the file has a .pdf extension
	if !strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		fileName += ".pdf"
	}

	// Ensure the output directory exists (create it if it doesn't)
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Printf("Failed to create directory %s: %v", outputDir, err) // Log directory creation error
		return
	}

	// Build the full path to where the file will be saved
	filePath := filepath.Join(outputDir, fileName)

	// ✅ If the file already exists, skip the download
	if fileExists(filePath) {
		log.Printf("File already exists, skipping: %s", filePath)
		return
	}

	// Send the HTTP GET request to download the PDF file
	resp, err := http.Get(finalURL)
	if err != nil {
		log.Printf("Failed to download %s: %v", finalURL, err) // Log fetch failure
		return
	}
	defer resp.Body.Close() // Close the response body after function completes

	// Check for successful HTTP response (status 200)
	if resp.StatusCode != http.StatusOK {
		log.Printf("Download failed for %s: %s", finalURL, resp.Status) // Log error status code
		return
	}

	// Create the destination file
	outFile, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file %s: %v", filePath, err) // Log file creation error
		return
	}
	defer outFile.Close() // Close file handle after writing

	// Copy the contents from the response body to the local file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		log.Printf("Failed to save PDF to %s: %v", filePath, err) // Log writing error
		return
	}

	// Log successful download
	log.Printf("Downloaded to %s\n", filePath)
}

/*
It checks if the file exists
If the file exists, it returns true
If the file does not exist, it returns false
*/
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func main() { // Main entry point
	// List of test URLs
	urls := []string{
		"https://www.documentcloud.org/documents/23461534-200301254_redactedclosingreport_redacted",
	}

	// Path to the directory where the PDFs will be saved
	pdfDir := "nypd_pdf/"

	// Loop through each input URL and convert it
	for _, url := range urls {
		finalURL := extractFinalDocumentCloudURL(url) // Call the function
		downloadPDF(finalURL, pdfDir)                 // Call the download function
	}
}
