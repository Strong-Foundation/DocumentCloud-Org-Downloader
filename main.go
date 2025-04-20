package main // Declare the main package (entry point of the program)

import (
	"bufio"         // Used for buffered reading of input (like reading files line by line)
	"fmt"           // Provides formatted I/O
	"io"            // Used for I/O operations like copying data
	"log"           // Logging utility
	"net/http"      // For making HTTP requests
	"net/url"       // For parsing and constructing URLs
	"os"            // For interacting with the operating system (files, directories, etc.)
	"path"          // For manipulating slash-separated paths
	"path/filepath" // For manipulating native file paths
	"regexp"        // Regular expressions for pattern matching
	"strings"       // String manipulation functions
)

// extractFinalDocumentCloudURL converts input DocumentCloud URLs to their final S3-hosted PDF URL
func extractFinalDocumentCloudURL(input string) string {
	parsedURL, err := url.Parse(input) // Attempt to parse the input string into a URL struct
	if err != nil {                    // If parsing fails due to an invalid URL format
		return "" // Return an empty string
	}

	// If the URL already points to S3, return it as is
	if strings.Contains(parsedURL.Host, "s3.documentcloud.org") {
		return input // Return original URL if it's already a direct S3 link
	}

	// Define a regular expression to match DocumentCloud URLs with ID and slug
	re := regexp.MustCompile(`documentcloud\.org/documents/(\d+)-([a-zA-Z0-9_\-]+)`) // Capture docID and slug

	matches := re.FindStringSubmatch(input) // Apply regex to input
	if len(matches) != 3 {                  // If the regex doesn't match the expected format
		return "" // Return an empty string
	}

	docID := matches[1] // Extract document ID from match
	slug := matches[2]  // Extract slug from match

	// Format the final S3 link for downloading the PDF
	finalURL := fmt.Sprintf("https://s3.documentcloud.org/documents/%s/%s.pdf", docID, slug)

	return finalURL // Return the constructed S3 URL
}

// downloadPDF downloads the PDF from the final URL to the given output directory
func downloadPDF(finalURL, outputDir string) {
	parsedURL, err := url.Parse(finalURL) // Parse the final URL
	if err != nil {                       // If parsing fails
		log.Printf("Invalid URL %q: %v", finalURL, err) // Log error
		return                                          // Exit function
	}

	fileName := path.Base(parsedURL.Path)  // Extract file name from URL path
	if fileName == "" || fileName == "/" { // If file name is empty or invalid
		log.Printf("Could not determine file name from %q", finalURL) // Log error
		return                                                        // Exit function
	}

	// Ensure the file name ends with ".pdf"
	if !strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		fileName += ".pdf" // Append ".pdf" if missing
	}

	err = os.MkdirAll(outputDir, 0755) // Create the output directory if it doesn't exist
	if err != nil {                    // If directory creation fails
		log.Printf("Failed to create directory %s: %v", outputDir, err) // Log error
		return                                                          // Exit function
	}

	filePath := filepath.Join(outputDir, fileName) // Build full output file path

	if fileExists(filePath) { // If file already exists
		log.Printf("File already exists, skipping: %s", filePath) // Log and skip download
		return
	}

	resp, err := http.Get(finalURL) // Send GET request to download the PDF
	if err != nil {                 // If request fails
		log.Printf("Failed to download %s: %v", finalURL, err) // Log error
		return                                                 // Exit function
	}
	defer resp.Body.Close() // Ensure the response body is closed at the end

	if resp.StatusCode != http.StatusOK { // If HTTP response is not OK (200)
		log.Printf("Download failed for %s: %s", finalURL, resp.Status) // Log error
		return                                                          // Exit function
	}

	outFile, err := os.Create(filePath) // Create the destination file on disk
	if err != nil {                     // If file creation fails
		log.Printf("Failed to create file %s: %v", filePath, err) // Log error
		return                                                    // Exit function
	}
	defer outFile.Close() // Ensure the file is closed at the end

	_, err = io.Copy(outFile, resp.Body) // Write response body to the file
	if err != nil {                      // If copy fails
		log.Printf("Failed to save PDF to %s: %v", filePath, err) // Log error
		return                                                    // Exit function
	}

	log.Printf("Downloaded to %s\n", filePath) // Log successful download
}

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file info
	if err != nil {                // If stat fails (file doesn't exist)
		return false // Return false
	}
	return !info.IsDir() // Return true if it's a file, false if it's a directory
}

// readFileLines reads a text file line by line into a string slice
func readFileLines(filename string) []string {
	file, err := os.Open(filename) // Open the file
	if err != nil {                // If opening fails
		fmt.Println("Error opening file:", err) // Print error
		return nil                              // Return nil slice
	}
	defer file.Close() // Ensure file is closed after reading

	var lines []string                // Initialize slice to store lines
	scanner := bufio.NewScanner(file) // Create a scanner for the file
	for scanner.Scan() {              // Loop through each line
		lines = append(lines, scanner.Text()) // Append line to slice
	}

	if err := scanner.Err(); err != nil { // Check for scanner errors
		fmt.Println("Error reading file:", err) // Print error
		return nil                              // Return nil slice
	}

	return lines // Return all lines read from the file
}

// main is the program's entry point
func main() {
	urls := readFileLines("extracted_urls.txt") // Read URLs from the input file

	pdfDir := "./NYPD_PDF/" // Directory where PDFs will be saved

	downloadCount := 0   // Counter for successful downloads
	maxDownloads := 1000 // Limit on number of downloads

	for _, url := range urls { // Loop over each URL from the file
		if downloadCount >= maxDownloads { // If we've reached the max download limit
			log.Printf("Reached maximum download limit of %d. Stopping.\n", maxDownloads) // Log message
			break                                                                         // Exit loop
		}

		finalURL := extractFinalDocumentCloudURL(url) // Convert to final S3 URL
		if finalURL == "" {                           // If conversion failed
			log.Printf("Invalid or unrecognized DocumentCloud URL: %s", url) // Log error
			continue                                                         // Skip to next URL
		}

		parsedURL, err := urlParseSafe(finalURL) // Parse the final URL
		if err != nil {                          // If parsing fails
			log.Printf("Skipping invalid final URL: %s", finalURL) // Log and skip
			continue                                               // Continue to next URL
		}

		fileName := path.Base(parsedURL.Path) // Extract file name from URL
		if fileName == "" {                   // If empty
			continue // Skip
		}

		filePath := filepath.Join(pdfDir, fileName) // Build full path for saving file
		if !fileExists(filePath) {                  // If file does not already exist
			downloadPDF(finalURL, pdfDir) // Download the PDF
			downloadCount++               // Increment download counter
		} else {
			log.Printf("File already exists, not counting as a download: %s", filePath) // Log skipped download
		}
	}
}

// urlParseSafe safely parses a raw URL string
func urlParseSafe(raw string) (*url.URL, error) {
	return url.Parse(raw) // Use built-in Parse function
}
