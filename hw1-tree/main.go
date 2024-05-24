package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

/*
go run main.go . -f
├───main.go (1881b)
├───main_test.go (1318b)
└───testdata
	├───project
	│	├───file.txt (19b)
	│	└───gopher.png (70372b)
	├───static
	│	├───css
	│	│	└───body.css (28b)
	│	├───html
	│	│	└───index.html (57b)
	│	└───js
	│		└───site.js (10b)
	├───zline
	│	└───empty.txt (empty)
	└───zzfile.txt (empty)
go run main.go .
└───testdata
	├───project
	├───static
	│	├───css
	│	├───html
	│	└───js
	└───zline
*/

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, files bool) error {
	open, err := os.Open(path)
	if err != nil {
		return err
	}

	stat, err := open.Stat()
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		_, err := fmt.Fprintf(out, "└───%s\n", buildFileName(stat))
		return err
	}

	return printTree(out, path, "", files)
}

func printTree(out io.Writer, dir string, indent string, files bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	if !files {
		removeFiles(&entries)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for i, entry := range entries {
		isLast := i == len(entries)-1
		prefix := "├───"
		nextIndent := indent + "│\t"
		if isLast {
			prefix = "└───"
			nextIndent = indent + "\t"
		}

		name := entry.Name()
		path := filepath.Join(dir, name)
		info, err := entry.Info()
		if err != nil {
			return err
		}

		if info.IsDir() {
			_, err := fmt.Fprintf(out, "%s%s%s\n", indent, prefix, name)
			if err != nil {
				return err
			}
			err = printTree(out, path, nextIndent, files)
			if err != nil {
				return err
			}
		} else {
			_, err := fmt.Fprintf(out, "%s%s\n", indent+prefix, buildFileName(info))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func buildFileName(fileInfo fs.FileInfo) string {
	if fileInfo.Size() == 0 {
		return fileInfo.Name() + " (empty)"
	} else {
		return fmt.Sprintf("%s (%db)", fileInfo.Name(), fileInfo.Size())
	}
}

func removeFiles(dirs *[]os.DirEntry) {
	last := 0
	for i := 0; i < len(*dirs); i++ {
		if (*dirs)[i].IsDir() {
			if i != last {
				(*dirs)[last] = (*dirs)[i]
			}
			last++
		}
	}
	*dirs = (*dirs)[:last]
}
