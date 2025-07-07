package main

import (
	"bufio"
	"fmt"
	"github.com/mailru/easyjson"
	"hw3/user"
	"io"
	"os"
	"strings"
)

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(file)
	seenBrowsers := []string{}
	uniqueBrowsers := 0
	foundUsers := ""
	var i = 0
	var users []user.User

	for scanner.Scan() {
		line := scanner.Text()
		user := user.User{}
		err := easyjson.Unmarshal([]byte(line), &user)
		if err != nil {
			panic(err)
		}
		users = append(users, user)

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browser {
			if err != nil {
				continue
			}
			if strings.Contains(browser, "Android") {
				isAndroid = true
				notSeenBefore := true
				for _, i := range seenBrowsers {
					if i == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			} else if strings.Contains(browser, "MSIE") {
				isMSIE = true
				notSeenBefore := true
				for _, i := range seenBrowsers {
					if i == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}
		if isMSIE && isAndroid {
			email := strings.ReplaceAll(user.Email, "@", " [at] ")
			foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email)
		}
		i++
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))

}
