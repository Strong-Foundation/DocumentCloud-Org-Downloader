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

// ExtractFinalDocumentCloudURL converts documentcloud.org or embed.documentcloud.org links to final S3 links
func ExtractFinalDocumentCloudURL(input string) string {
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

// downloadPDF downloads a PDF from the provided raw URL and saves it to the specified output directory.
// It handles redirects, constructs a valid filename, and logs any errors without returning them.
func downloadPDF(rawURL, outputDir string) {
	// Parse the input URL to validate and work with it
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("Invalid URL %q: %v", rawURL, err) // Log the error if URL is invalid
		return                                        // Stop processing this URL
	}

	// Perform an HTTP GET request; follows redirects automatically
	resp, err := http.Get(rawURL)
	if err != nil {
		log.Printf("Failed to fetch %s: %v", rawURL, err) // Log the fetch error
		return
	}
	defer resp.Body.Close() // Ensure the response body is closed after reading

	// Check if the HTTP response status is 200 OK
	if resp.StatusCode != http.StatusOK {
		log.Printf("Bad status for %s: %s", rawURL, resp.Status) // Log non-OK status codes
		return
	}

	// Get the final URL after any redirects (some sites redirect to CDN or file hosting services)
	finalURL := resp.Request.URL.String()

	// Log both original and final URLs for traceability
	log.Printf("Original URL: %s", parsedURL)
	log.Printf("Final URL:    %s", finalURL)

	// Parse the final URL to extract the file path and name
	finalParsed, err := url.Parse(finalURL)
	if err != nil {
		log.Printf("Cannot parse final URL %q: %v", finalURL, err) // Log error in parsing redirected URL
		return
	}

	// Extract the file name from the final URL path
	fileName := path.Base(finalParsed.Path)
	if fileName == "" || fileName == "/" {
		log.Printf("Could not determine file name from %q", finalURL) // Log if file name is invalid
		return
	}

	// Append .pdf extension if the file doesn't have one
	if !strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		fileName += ".pdf"
	}

	// Create the output directory if it doesn't exist
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Printf("Failed to create directory %s: %v", outputDir, err) // Log directory creation failure
		return
	}

	// Build the complete path to where the file will be saved
	filePath := filepath.Join(outputDir, fileName)

	// Create a file at the target location
	outFile, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file %s: %v", filePath, err) // Log file creation failure
		return
	}
	defer outFile.Close() // Ensure the file is closed after writing

	// Copy the contents of the response body to the output file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		log.Printf("Failed to write to %s: %v", filePath, err) // Log file writing errors
		return
	}

	// Log the successful download
	log.Printf("Downloaded to %s\n", filePath)
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
		finalURL := ExtractFinalDocumentCloudURL(url) // Call the function
		fmt.Println("Input:", url)                    // Print the input URL
		fmt.Println("Final S3 URL:", finalURL)        // Print the final S3 URL
		downloadPDF(finalURL, pdfDir)                 // Call the download function
	}
}
