package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

func findMarkdownFiles(rootDir string) ([]string, error) {
	var markdownFiles []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			markdownFiles = append(markdownFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return markdownFiles, nil
}

func extractLinksFromFile(filePath string) ([][3]string, error) {
	var links [][3]string
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 1
	re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				links = append(links, [3]string{match[1], filePath, fmt.Sprintf("%d", lineNum)})
			}
		}
		lineNum++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return links, nil
}

func getAllLinks(markdownFiles []string) (map[string][][2]string, error) {
	allLinks := make(map[string][][2]string)
	for _, file := range markdownFiles {
		links, err := extractLinksFromFile(file)
		if err != nil {
			return nil, err
		}
		for _, link := range links {
			allLinks[link[0]] = append(allLinks[link[0]], [2]string{link[1], link[2]})
		}
	}
	return allLinks, nil
}

func findExistingFiles(rootDir string) (map[string][]string, error) {
	existingFiles := make(map[string][]string)
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			fileName := strings.TrimSuffix(filepath.Base(path), ".md")
			relativePath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err
			}
			existingFiles[fileName] = append(existingFiles[fileName], relativePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return existingFiles, nil
}

func findMissingFiles(rootDir string, allLinks map[string][][2]string) (map[string][][2]string, error) {
	existingFiles, err := findExistingFiles(rootDir)
	if err != nil {
		return nil, err
	}
	missingFiles := make(map[string][][2]string)
	for link, locations := range allLinks {
		if _, found := existingFiles[link]; !found {
			missingFiles[link] = locations
		}
	}
	return missingFiles, nil
}

func createMissingFiles(rootDir string, missingFiles map[string][][2]string) error {
	for link := range missingFiles {
		filePath := filepath.Join(rootDir, link+".md")
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}

func main() {
	fmt.Print("Enter the root directory of the Obsidian vault: ")
	var rootDir string
	fmt.Scanln(&rootDir)

	markdownFiles, err := findMarkdownFiles(rootDir)
	if err != nil {
		fmt.Println("Error finding markdown files:", err)
		return
	}

	allLinks, err := getAllLinks(markdownFiles)
	if err != nil {
		fmt.Println("Error extracting links from files:", err)
		return
	}

	missingFiles, err := findMissingFiles(rootDir, allLinks)
	if err != nil {
		fmt.Println("Error finding missing files:", err)
		return
	}

	if len(missingFiles) > 0 {
		fmt.Println("Missing files:")
		for link, locations := range missingFiles {
			fmt.Printf("%s is referenced in:\n", color.RedString(link+".md"))
			for _, location := range locations {
				fmt.Printf("  - %s (line %s)\n", color.GreenString(location[0]), location[1])
			}
		}

		fmt.Print("Do you want to create these missing files? (y/n): ")
		var create string
		fmt.Scanln(&create)
		if strings.ToLower(create) == "y" {
			if err := createMissingFiles(rootDir, missingFiles); err != nil {
				fmt.Println("Error creating missing files:", err)
				return
			}
			fmt.Println("Missing files created.")
		} else {
			fmt.Println("No files were created.")
		}
	} else {
		fmt.Println("No missing files found.")
	}
}

