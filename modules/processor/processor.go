package processor

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"sgrep/modules/model"
	"sgrep/modules/similarity"
	"sgrep/modules/utils"

	"github.com/clipperhouse/uax29/words"
)

func ProcessLineByLine(query string, w2vModel model.VectorModel, similarityCache similarity.SimilarityCache,
	similarityThreshold float64, contextBefore, contextAfter int, input *os.File,
	printLineNumbers, ignoreCase, outputOnlyMatching, outputOnlyLines bool) {

	// Prepare query vector
	var queryTokenToCheck string
	if ignoreCase {
		queryTokenToCheck = strings.ToLower(query)
	} else {
		queryTokenToCheck = query
	}
	queryVector := w2vModel.GetEmbedding(queryTokenToCheck)

	scanner := bufio.NewScanner(input)
	lineNumber := 0
	var contextBuffer []string
	var contextLineNumbers []int

	// Process each line
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		matched := false
		var highlightedLine string
		var similarityScore float64

		// Tokenize and check each token
		tokens := words.NewSegmenter(scanner.Bytes())
		for tokens.Next() {
			token := tokens.Text()
			var tokenToCheck string
			if ignoreCase {
				tokenToCheck = strings.ToLower(token)
			} else {
				tokenToCheck = token
			}

			// Calculate similarity and check threshold
			tokenVector := w2vModel.GetEmbedding(tokenToCheck)
			similarity := similarityCache.MemoizedCalculateSimilarity(queryTokenToCheck, tokenToCheck, queryVector, tokenVector)
			if similarity > similarityThreshold {
				highlightedLine = strings.Replace(line, token, utils.ColorText(token, "red"), -1)
				matched = true
				similarityScore = similarity
				if outputOnlyMatching {
					fmt.Println(token)
					break // Stop after first match if -o is set
				}
			}
		}

		// Handle matched line
		if matched {
			if outputOnlyMatching {
				// Already printed in the loop above
			} else if outputOnlyLines {
				utils.PrintLine(highlightedLine, lineNumber, printLineNumbers)
			} else {
				fmt.Printf("Similarity: %.4f\n", similarityScore)
				// Print the context lines before the match
				for i, ctxLine := range contextBuffer {
					utils.PrintLine(ctxLine, contextLineNumbers[i], printLineNumbers)
				}

				// Print the matched line with highlighted token
				utils.PrintLine(highlightedLine, lineNumber, printLineNumbers)

				// Print the context lines after the match
				for i := 0; i < contextAfter && scanner.Scan(); i++ {
					lineNumber++
					utils.PrintLine(scanner.Text(), lineNumber, printLineNumbers)
				}

				fmt.Println("--")
			}

			// Clear the context buffer after printing
			contextBuffer = nil
			contextLineNumbers = nil
		} else {
			// Update the context buffer with the current line if no match is found
			if contextBefore > 0 && !outputOnlyMatching && !outputOnlyLines {
				contextBuffer = append(contextBuffer, line)
				contextLineNumbers = append(contextLineNumbers, lineNumber)
				// Ensure the context buffer does not exceed the specified number of lines
				if len(contextBuffer) > contextBefore {
					contextBuffer = contextBuffer[1:]
					contextLineNumbers = contextLineNumbers[1:]
				}
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}
