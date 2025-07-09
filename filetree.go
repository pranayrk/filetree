package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/urfave/cli/v3"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"os/exec"
)

type MapEntry struct {
	IsFolder bool
	Name     string
	Path     string
	Tags     map[string]bool
}

func crash(err error, text string) {
	if err != nil {
		if text != "" {
			err = fmt.Errorf("Error: %q\n %w", text, err)
		}
		log.Fatal(err)
	}
}

func parseTags(line string) (string, map[string]bool) {
	line = strings.TrimSpace(line)

	openBracket := strings.LastIndex(line, "[")
	closeBracket := strings.LastIndex(line, "]")

	if openBracket == -1 || closeBracket == -1 || openBracket > closeBracket {
		return line, nil
	}

	name := strings.TrimSpace(line[:openBracket])
	tagsStr := line[openBracket+1 : closeBracket]
	tags := make(map[string]bool)
	for _, tag := range strings.Split(tagsStr, ",") {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			tags[trimmed] = true
		}
	}

	return name, tags
}

func addMissingFiles(mapFile string, folderPath string) {
	var existingEntries []MapEntry
	if _, err := os.Stat(filepath.Join(folderPath, mapFile)); err == nil {
		existingEntries = readMap(mapFile, folderPath, nil, false)
	}
	existingNames := make(map[string]bool)
	for _, entry := range existingEntries {
		existingNames[entry.Name] = true
	}

	// Scan the actual folder directory
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		fmt.Println("Error reading directory: " + folderPath + "\n")
		return
	}

	// Open map file for appending
	file, err := os.OpenFile(filepath.Join(folderPath, mapFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	crash(err, "Error opening map file for appending")
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, entry := range entries {
		// Skip directories (only add files)
		if entry.IsDir() {
			continue
		}
		// Skip the map file itself
		if entry.Name() == filepath.Base(mapFile) {
			continue
		}
		// Skip if already in map
		if existingNames[entry.Name()] {
			continue
		}
		// Add missing file to map
		line := fmt.Sprintf("> %s\n", entry.Name())
		fmt.Println("Adding > " + entry.Name())
		writer.WriteString(line)
	}

	writer.Flush()
}

func cleanup(mapFile string, basePath string) {
	if _, err := os.Stat(filepath.Join(basePath, mapFile)); err != nil {
		fmt.Println("Could not find mapFile " + mapFile + " at path" + basePath + " during cleanup")
		return
	}

	entries := readMap(mapFile, basePath, nil, false)
	var validEntries []MapEntry

	for _, entry := range entries {
		if _, err := os.Stat(entry.Path); err == nil {
			validEntries = append(validEntries, entry)
		}
	}

	file, err := os.Create(filepath.Join(basePath, mapFile))
	crash(err, "Error creating map file for cleanup")
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range validEntries {
		var prefix string
		if entry.IsFolder {
			prefix = ">>"
		} else {
			prefix = ">"
		}
		line := fmt.Sprintf("%s %s", prefix, entry.Name)
		if len(entry.Tags) > 0 {
			tagList := make([]string, 0, len(entry.Tags))
			for tag := range entry.Tags {
				tagList = append(tagList, tag)
			}
			line += fmt.Sprintf(" [%s]", strings.Join(tagList, ", "))
		}
		line += "\n"
		writer.WriteString(line)
	}
	writer.Flush()
}

func readMap(mapFile string, basePath string, entries []MapEntry, recurse bool) []MapEntry {
	if _, err := os.Stat(filepath.Join(basePath, mapFile)); err != nil {
		fmt.Println("No map file found at:\n" + mapFile + "\nSkipping...\n")
		return entries
	}
	if entries == nil {
		entries = []MapEntry{}
	}

	file, err := os.Open(filepath.Join(basePath,mapFile))
	crash(err, "Error reading map file")
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		isFolder := false
		var rest string
		if strings.HasPrefix(line, ">>") {
			rest = strings.TrimSpace(strings.TrimPrefix(line, ">>"))
			isFolder = true
		} else if strings.HasPrefix(line, ">") {
			rest = strings.TrimSpace(strings.TrimPrefix(line, ">"))
		} else {
			continue
		}
		name, tags := parseTags(rest)
		path := filepath.Join(basePath, name)
		if tags == nil {
			tags = make(map[string]bool)
		}
		entries = append(entries, MapEntry{IsFolder: isFolder, Name: name, Path: path, Tags: tags})
		if isFolder && recurse {

			nestedMapFile := name +".map"
			entries = readMap(nestedMapFile, path, entries, recurse)
		}
	}
	return entries
}

func refresh(ctx context.Context, cmd *cli.Command) error {
	arg := cmd.Args().First()
	if arg != "" {
		err := os.Chdir(arg)
		crash(err, "Error changing directory")
	}

	cwd, err := os.Getwd()
	crash(err, "Error getting current working directory")

	folderName := filepath.Base(cwd)
	mapFile := folderName + ".map"

	_, err = os.Stat(mapFile)
	crash(err, "Map file not found")

	// Clean up stale entries from the main map file
	cleanup(mapFile, cwd)
	addMissingFiles(mapFile, cwd)

	entries := readMap(mapFile, cwd, nil, true)

	// Sync and cleanup for each subfolder
	for _, entry := range entries {
		if entry.IsFolder {
			folderMapFile := entry.Name + ".map"
			cleanup(folderMapFile, entry.Path)
			addMissingFiles(folderMapFile, entry.Path)
		}
	}

	return nil
}

func exportFiles(ctx context.Context, cmd *cli.Command) error {
	exportFolder := cmd.Args().First()
	if exportFolder == "" {
		crash(fmt.Errorf("Provide an output folder to export files"), "")
	}

	cwd, err := os.Getwd()
	crash(err, "Error getting current working directory")

	folderName := filepath.Base(cwd)
	mapFile := folderName + ".map"

	_, err = os.Stat(mapFile)
	crash(err, "Map file not found")

	// Empty the export folder before exporting (if it exists)
	_ = os.RemoveAll(exportFolder)
	err = os.MkdirAll(exportFolder, 0755)
	crash(err, "Error recreating export folder")

	entries := readMap(mapFile, cwd, nil, true)

	// Export files for each file entry
	for _, entry := range entries {
		if !entry.IsFolder {
			if len(entry.Tags) == 0 {
				fmt.Printf("Skipping %s (no tags)\n", entry.Name)
				continue
			}

			// Export to each tag folder (tags may contain "/" for nesting)
			for tag := range entry.Tags {
				tagPath := filepath.Join(exportFolder, tag)

				err = os.MkdirAll(tagPath, 0755)
				crash(err, "Error creating tag folder: "+tagPath)

				destPath := filepath.Join(tagPath, entry.Name)

				// Check if copy or symlink (default: symlink)
				useCopy := cmd.String("type") == "copy"

				if useCopy {
					err = copyFile(entry.Path, destPath)
					if err != nil {
						fmt.Printf("Error copying %s to %s: %v\n", entry.Name, destPath, err)
						continue
					}
					fmt.Printf("Copied: %s -> %s\n", entry.Path, destPath)
				} else {
					err = os.Symlink(entry.Path, destPath)
					if err != nil {
						fmt.Printf("Error creating symlink for %s in %s: %v\n", entry.Name, tag, err)
						continue
					}
					fmt.Printf("Created symlink: %s -> %s\n", destPath, entry.Path)
				}
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func writeMap(mapFile string, dir string, entries []MapEntry) error {
	file, err := os.Create(filepath.Join(dir, mapFile))
	crash(err, "Error creating map File")
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		var prefix string
		if entry.IsFolder {
			prefix = ">>"
		} else {
			prefix = ">"
		}
		line := fmt.Sprintf("%s %s", prefix, entry.Name)
		if len(entry.Tags) > 0 {
			tagList := make([]string, 0, len(entry.Tags))
			for tag := range entry.Tags {
				tagList = append(tagList, tag)
			}
			line += fmt.Sprintf(" [%s]", strings.Join(tagList, ", "))
		}
		line += "\n"
		writer.WriteString(line)
	}
	return writer.Flush()
}

func previewFile(path string) {
	cmd := exec.Command("xdg-open", path)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Failed to open file: %s", path)
	}
}

func interactive(ctx context.Context, cmd *cli.Command) error {
	arg := cmd.Args().First()
	if arg != "" {
		err := os.Chdir(arg)
		crash(err, "Error changing directory")
	}

	cwd, err := os.Getwd()
	crash(err, "Error getting current working directory")

	folderName := filepath.Base(cwd)
	mapFile := folderName + ".map"
	if _, err := os.Stat(mapFile); err != nil {
		crash(fmt.Errorf("Map file not found: %s", mapFile), "")
	}

	// Run refresh before interactive tagging
	refresh(ctx, cmd)

	untaggedOnly := cmd.Bool("untagged")
	tagFiles(mapFile, cwd, untaggedOnly)
	return nil
}

func tagFiles(mapFile string, dir string, untaggedOnly bool) {
	entries := readMap(mapFile, dir, nil, false)

	reader := bufio.NewReader(os.Stdin)

	for i, entry := range entries {
		if entry.IsFolder {
			tagFiles(entry.Name + ".map", filepath.Join(dir, entry.Name), untaggedOnly)
			continue
		}

		if untaggedOnly && len(entry.Tags) > 0 {
			continue
		}

		fmt.Printf("\n=== File: %s ===\n", entry.Name)

		if len(entry.Tags) > 0 {
			tagList := make([]string, 0, len(entry.Tags))
			for tag := range entry.Tags {
				tagList = append(tagList, tag)
			}
			fmt.Printf("Current tags: [%s]\n", strings.Join(tagList, ", "))
		} else {
			fmt.Println("Current tags: (none)")
		}

		fmt.Println("Press 'p' to preview, 'q' to quit, or enter tags: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input, skipping...")
			continue
		}
		input = strings.TrimSpace(input)

		if input == "p" {
			previewFile(entry.Path)
			fmt.Println("Press 'q' to quit, or enter tags: ")
			input, err = reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading input, skipping...")
				continue
			}
			input = strings.TrimSpace(input)
		}

		if input == "q" {
			fmt.Println("Exiting tag command.")
			break
		}

		if input == "s" {
			fmt.Println("Skipped.")
			continue
		}

		if input != "" {
			if entry.Tags == nil {
				entry.Tags = make(map[string]bool)
			}
			for _, tag := range strings.Split(input, ",") {
				trimmed := strings.TrimSpace(tag)
				if trimmed != "" {
					entry.Tags[trimmed] = true
				}
			}
			entries[i] = entry
			tagList := make([]string, 0, len(entry.Tags))
			for tag := range entry.Tags {
				tagList = append(tagList, tag)
			}
			fmt.Printf("Tags updated: [%s]\n", strings.Join(tagList, ", "))
		}

		err = writeMap(mapFile, dir, entries)
		if err != nil {
			fmt.Printf("Error writing map file: %v\n", err)
		}
	}
}

func main() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	addCommand := func(commands []*cli.Command, name string, alias string, usage string,
	action func(ctx context.Context, cmd *cli.Command) error) []*cli.Command {
		commands = append(commands, &cli.Command{
			Name:    name,
			Aliases: []string{alias},
			Usage:   usage,
			Action:  action,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "type",
					Aliases: []string{"t"},
					Usage:   "Export type: copy or symlink (default: symlink)",
					Value:   "symlink",
				},
			},
		})
		return commands
	}

	var commands []*cli.Command
	commands = addCommand(commands, "refresh", "r", "Update all folders and cleanup deleted files", refresh)
	commands = addCommand(commands, "export", "x", "Export files from tags (use -t copy for copying files)", exportFiles)
	commands = append(commands, &cli.Command{
		Name:    "interactive",
		Aliases: []string{"i"},
		Usage:   "Interactively tag files one by one",
		Action:  interactive,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "untagged",
				Aliases: []string{"u"},
				Usage:   "Only show files without any tags",
			},
		},
	})

	cmd := &cli.Command{
		Name:   "filetree",
		Usage:  "A File Organizer",
		Commands: commands,
	}

	e := cmd.Run(context.Background(), os.Args)

	crash(e, "")
}
