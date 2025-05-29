package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"sort"
	"strconv"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage: go run main.go <path> [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"

	if err := dirTree(out, path, printFiles); err != nil {
		panic(err)
	}
}

func dirTree(out io.Writer, path string, printFiles bool) (err error) {
	return printer(out, path, printFiles, "")
}

func printer(out io.Writer, path string, printFiles bool, prefix string) (err error) {
	catalog, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	var packages []fs.DirEntry
	var files []fs.DirEntry
	var separator string

	for _, item := range catalog {
		if item.IsDir() {
			packages = append(packages, item)
		} else {
			files = append(files, item)
		}
	}

	var packagesAndFiles = append(packages, files...)

	sort.Slice(packagesAndFiles, func(i, j int) bool {
		return packagesAndFiles[i].Name() < packagesAndFiles[j].Name()
	})

	for i, item := range packagesAndFiles {
		var fileName = item.Name()
		separator += "├───"
		if i == len(packagesAndFiles)-1 {
			separator += "└───"
		}
		if item.IsDir() {
			newPrefix := prefix
			if i == len(packagesAndFiles)-1 {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			err := printer(out, path+"/"+item.Name(), printFiles, newPrefix)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			info, err := item.Info()
			if err != nil {
				log.Fatal(err)
			}
			var size = info.Size()
			if size == 0 {
				fileName += " (empty)"
			} else {
				fileName += "(" + strconv.Itoa(int(size)) + "b)"
			}
		}

		fmt.Fprintf(out, "%s%s%s\n", prefix, separator, fileName)
	}
	return nil
}
